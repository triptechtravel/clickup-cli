package task

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/triptechtravel/clickup-cli/internal/taskindex"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

// spaceTree wires a minimal space → folderless-list → tasks hierarchy, with a
// per-list page table so pagination and failures can be exercised.
func spaceTree(tf *testutil.TestFactory, listPages map[string]map[int]string, failing map[string]bool) *[]string {
	var mu sync.Mutex
	var urls []string

	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[{"id":"sp1","name":"Space One"}]}`)
	tf.Handle("GET", "space/sp1/folder", 200, `{"folders":[]}`)

	ids := make([]string, 0, len(listPages))
	for id := range listPages {
		ids = append(ids, fmt.Sprintf(`{"id":%q,"name":"List %s"}`, id, id))
	}
	tf.Handle("GET", "space/sp1/list", 200, `{"lists":[`+strings.Join(ids, ",")+`]}`)

	for listID, pages := range listPages {
		lid, pgs := listID, pages
		tf.HandleFunc("list/"+lid+"/task", func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			urls = append(urls, r.URL.String())
			mu.Unlock()

			if failing[lid] {
				w.WriteHeader(500)
				_, _ = w.Write([]byte(`{"err":"boom"}`))
				return
			}
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			body := `{"tasks":[]}`
			if b, ok := pgs[page]; ok {
				body = b
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Remaining", "99")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(body))
		})
	}
	return &urls
}

// The space walk is advertised as the exhaustive path — the truncation notice
// and the help text both send users to it — but it read only page 0 of each
// list, so anything past the first hundred tasks in a list was unreachable by
// the one route promised to reach everything.
func TestSpaceWalk_PaginatesEachList(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	urls := spaceTree(tf, map[string]map[int]string{
		"L1": {
			0: searchTasksJSON(fillerPairs(100)...),
			1: searchTasksJSON([2]string{"deep", "Buried 5.6.1 card"}),
		},
	}, nil)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--space", "Space One", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.OutBuf.String(), "Buried 5.6.1 card", "list not paginated past page 0")
	var sawPage1 bool
	for _, u := range *urls {
		if strings.Contains(u, "page=1") {
			sawPage1 = true
		}
	}
	assert.True(t, sawPage1)
}

// A list that cannot be read contributed nothing and said nothing, so a
// partial walk was indistinguishable from a complete one.
func TestSpaceWalk_DisclosesListsItCouldNotRead(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	spaceTree(tf, map[string]map[int]string{
		"L1": {0: searchTasksJSON([2]string{"a", "5.6.1 found"})},
		"L2": {0: `{"tasks":[]}`},
	}, map[string]bool{"L2": true})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--space", "Space One", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOut := tf.ErrBuf.String()
	assert.Contains(t, errOut, "Not shown:", "unreadable list passed over in silence")
	assert.Contains(t, errOut, "could only be read partway",
		"a list read partway was reported with the wrong mechanism: %s", errOut)
	assert.NotContains(t, errOut, "hold more than",
		"per-list failure described as the page cap")
}

// A capped live sweep that then finds nothing in the tree walk must still say
// the sweep was capped. Dropping it turns a partial search into a confident
// "no such task".
func TestSearch_LiveFallbackKeepsTheSweepDisclosure(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	full := searchTasksJSON(fillerPairs(100)...)
	pages := map[int]string{}
	for i := 0; i < maxSweepPages+1; i++ {
		pages[i] = full
	}
	sweepMux(tf, pages, `{"tasks":[]}`)
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "nomatchhere", "--no-cache", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.ErrBuf.String(), "Not shown:", "sweep truncation lost in the fallback")
}

// The disclosure has to name the mechanism that actually bit. Telling someone
// the "10-page cap (~1000 tasks)" stopped a search that in fact hit the index
// sync budget sends them after the wrong remedy.
func TestSearch_IndexTruncationNamesTheIndexNotTheSweep(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	idx := taskindex.New("12345")
	idx.MergePartial([]taskindex.Entry{
		{ID: "a", Name: "5.6.1 card", DateUpdated: 1_000_000},
	}, time.Now())
	if err := taskindex.Save(tf.CacheDir, idx); err != nil {
		t.Fatal(err)
	}
	recordingMux(tf, nil)
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOut := tf.ErrBuf.String()
	assert.Contains(t, errOut, "Not shown:")
	assert.Contains(t, errOut, "index")
	assert.NotContains(t, errOut, fmt.Sprintf("%d-page cap", maxSweepPages),
		"index truncation described as the live sweep's page cap")
}

// A transient 5xx must not be recorded as a finished rebuild. It keeps the
// pages that arrived, leaves a resume point, and does not stamp the reconcile
// clock — so the next search continues rather than writing the workspace off
// for a week, which is what treating an error as budget exhaustion did.
func TestSearch_SyncErrorLeavesAResumePointNotAFinishedBuild(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir

	tf.HandleFunc("team/12345/task", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "0" {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"err":"boom"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(searchTasksJSON(append(fillerPairs(99), [2]string{"A", "5.6.1 card"})...)))
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reloaded, _ := taskindex.Load(dir, "12345")
	assert.Contains(t, reloaded.Entries, "A", "pages read before the error were discarded")
	assert.True(t, reloaded.NeedsFullSync(time.Now(), cacheTTL),
		"a failed build was recorded as finished, so it will not be retried")
	assert.NotZero(t, reloaded.RebuildFloor, "no resume point, so the next run restarts from scratch")
}

// An index that has never held anything must be rebuilt, not topped up, and a
// clean rebuild must clear the incomplete flag.
func TestSearch_EmptyIndexIsRebuiltAndClearsPartial(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	// What a failed first run leaves behind: flagged partial, nothing in it.
	idx := taskindex.New("12345")
	idx.MergePartial(nil, time.Now())
	if err := taskindex.Save(dir, idx); err != nil {
		t.Fatal(err)
	}
	urls := recordingMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"a", "5.6.1 card"}),
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.False(t, anyContains(*urls, "date_updated_gt"), "topped up an index holding nothing")

	reloaded, _ := taskindex.Load(dir, "12345")
	assert.False(t, reloaded.Partial, "clean rebuild left the index flagged incomplete")
	assert.NotContains(t, tf.ErrBuf.String(), "Not shown:")
}

// A truncated incremental holds its watermark, so on its own it would repeat
// the same capped read forever. It has to ask for a rebuild to make progress.
func TestSearch_TruncatedIncrementalSchedulesARebuild(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{{ID: "seed", Name: "Seed", DateUpdated: 1_000_000}}, time.Now())
	if err := taskindex.Save(dir, idx); err != nil {
		t.Fatal(err)
	}

	full := searchTasksJSON(fillerPairs(100)...)
	pages := map[int]string{}
	for i := 0; i < maxFullSyncPages+1; i++ {
		pages[i] = full
	}
	recordingMux(tf, pages)
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "nothingmatchesthis", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reloaded, _ := taskindex.Load(dir, "12345")
	assert.Equal(t, int64(1_000_000), reloaded.SyncedAt, "watermark advanced over an unread gap")
	assert.True(t, reloaded.NeedsFullSync(time.Now(), cacheTTL),
		"capped incremental left no way to ever catch up")
}

// Splitting a query into words must not re-sync the index once per word, nor
// launch a tree walk per word.
func TestSearch_WordRetryDoesNotResyncPerWord(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{
		{ID: "a", Name: "offline database work", DateUpdated: 1_000_000},
	}, time.Now())
	if err := taskindex.Save(tf.CacheDir, idx); err != nil {
		t.Fatal(err)
	}
	urls := recordingMux(tf, nil)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "offline sqlite upgrade"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Len(t, *urls, 1, "re-synced the index once per word")
	assert.Contains(t, tf.OutBuf.String(), "offline database work")
}

// A deadline is not a permission problem. Counting every list left unvisited
// when the clock ran out as "could not be read" reported 127 broken lists when
// the truth was one expired budget — the wrong-mechanism failure again, one
// level down.
//
// The first version of this test cancelled the context up front, which made the
// walk bail during space discovery and never reach the per-list loop it claims
// to exercise; the guard could be deleted and the test stayed green. Discovery
// now succeeds and the deadline expires inside the list fetch.
func TestSpaceWalk_DeadlineIsNotReportedAsUnreadableLists(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[{"id":"sp1","name":"Space One"}]}`)
	tf.Handle("GET", "space/sp1/folder", 200, `{"folders":[]}`)
	tf.Handle("GET", "space/sp1/list", 200, `{"lists":[{"id":"L1","name":"L1"},{"id":"L2","name":"L2"}]}`)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var reached atomic.Bool
	for _, id := range []string{"L1", "L2"} {
		tf.HandleFunc("list/"+id+"/task", func(w http.ResponseWriter, r *http.Request) {
			// The walk got as far as fetching tasks; now the clock runs out.
			reached.Store(true)
			cancel()
			<-r.Context().Done()
		})
	}

	opts := &searchOptions{factory: tf.Factory, query: "5.6.1", space: "Space One"}
	res, err := searchViaSpaces(ctx, opts)

	assert.NoError(t, err)
	assert.True(t, reached.Load(), "test never reached the per-list fetch it is about")
	assert.True(t, res.cancelled, "cancellation not reported")
	for _, n := range res.notShown {
		assert.NotContains(t, n, "could not be read",
			"a cancelled walk blamed the lists: %s", n)
	}
}

// Every indexed field has to survive the round trip, or the same query returns
// different JSON depending on whether the cache happened to be warm. Asserted
// through the command, because in-memory assertions on the struct stayed green
// when the projection dropped fields.
func TestSearch_CachedResultsCarryEveryProjectedField(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/task", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		if r.URL.Query().Get("page") != "0" {
			_, _ = w.Write([]byte(`{"tasks":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"tasks":[{
			"id":"A","custom_id":"CU-A","name":"Offline DB upgrade",
			"description":"replace op-sqlite with the maintained fork",
			"status":{"status":"in progress"},"priority":{"priority":"high"},
			"assignees":[{"username":"isaac"}],"url":"https://app.clickup.com/t/A",
			"parent":"","date_updated":"1700000000000"},{
			"id":"B","name":"Offline DB unassigned","status":{"status":"open"},
			"assignees":[],"date_updated":"1700000000001"}]}`))
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "Offline", "--exact", "--json"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(tf.OutBuf.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, tf.OutBuf.String())
	}
	assert.Len(t, got, 2)
	// Newest first, so the unassigned task leads. It is the one that matters
	// here: with an assignee present the slice is non-nil either way, which is
	// why the first version of this assertion could not fail.
	assert.NotNil(t, got[0]["assignees"], "unassigned task came back null, not []")
	assert.Len(t, got[0]["assignees"], 0)

	assert.Equal(t, "CU-A", got[1]["custom_id"])
	assert.Equal(t, "replace op-sqlite with the maintained fork", got[1]["description"])
	assert.Equal(t, "high", got[1]["priority"].(map[string]any)["priority"])
	assert.Equal(t, "in progress", got[1]["status"].(map[string]any)["status"])
	assert.Equal(t, "https://app.clickup.com/t/A", got[1]["url"])
	assert.Equal(t, "1700000000000", got[1]["date_updated"])
	assert.Len(t, got[1]["assignees"], 1)
}

// The command matches on descriptions as well as names, and the index has to
// preserve that. Every other cache fixture matches on Name, so dropping the
// description from the projection went unnoticed.
func TestSearch_IndexPreservesDescriptionMatching(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	recordingMux(tf, map[int]string{
		0: `{"tasks":[{"id":"A","name":"Unrelated title","description":"mentions 5.6.1 in the body","status":{"status":"open"},"assignees":[],"date_updated":"9000000"}]}`,
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.OutBuf.String(), "Unrelated title",
		"description match lost in the index round trip")
}

// The incremental sync must ask from the overlap window, not the raw
// watermark. Asserting only that the param is present let the overlap — whose
// absence the index docs call permanent task loss — be deleted silently.
func TestSearch_IncrementalAsksFromTheOverlappedWatermark(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{{ID: "a", Name: "A", DateUpdated: 5_000_000}}, time.Now())
	if err := taskindex.Save(tf.CacheDir, idx); err != nil {
		t.Fatal(err)
	}
	urls := recordingMux(tf, nil)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "anything"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := fmt.Sprintf("date_updated_gt=%d", idx.Since())
	assert.True(t, anyContains(*urls, want),
		"expected %s, got %v", want, *urls)
}

// Results come out of a map, so without an explicit sort the order changes run
// to run. Pinned by asserting the order rather than hoping to observe flake.
func TestSearch_IndexResultsAreOrderedNewestFirst(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{
		{ID: "old", Name: "5.6.1 older card", DateUpdated: 1_000},
		{ID: "new", Name: "5.6.1 newer card", DateUpdated: 9_000},
	}, time.Now())
	if err := taskindex.Save(tf.CacheDir, idx); err != nil {
		t.Fatal(err)
	}
	recordingMux(tf, nil)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := tf.OutBuf.String()
	assert.Less(t, strings.Index(out, "newer card"), strings.Index(out, "older card"),
		"index results not ordered newest first")
}
