package task

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
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

	assert.Contains(t, tf.ErrBuf.String(), "Not shown:", "unreadable list passed over in silence")
	assert.Contains(t, tf.ErrBuf.String(), "1 list")
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
// when the clock ran out as "could not be read" reports 127 broken lists when
// the truth is one expired budget — the wrong-mechanism failure again, one
// level down.
func TestSpaceWalk_DeadlineIsNotReportedAsUnreadableLists(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	pages := map[string]map[int]string{}
	for _, id := range []string{"L1", "L2", "L3"} {
		pages[id] = map[int]string{0: `{"tasks":[]}`}
	}
	spaceTree(tf, pages, nil)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	opts := &searchOptions{factory: tf.Factory, query: "5.6.1", space: "Space One"}
	res, err := searchViaSpaces(ctx, opts)

	assert.NoError(t, err)
	assert.True(t, res.cancelled, "cancellation not reported")
	for _, n := range res.notShown {
		assert.NotContains(t, n, "could not be read",
			"a cancelled walk blamed the lists: %s", n)
	}
}
