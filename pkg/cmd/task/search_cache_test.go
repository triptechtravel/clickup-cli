package task

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/triptechtravel/clickup-cli/internal/taskindex"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

// recordingMux serves the team task endpoint from a per-page table and records
// every request URL, so tests can assert on what was actually asked for.
func recordingMux(tf *testutil.TestFactory, pages map[int]string) *[]string {
	var urls []string
	var mu sync.Mutex
	tf.HandleFunc("team/12345/task", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		urls = append(urls, r.URL.String())
		mu.Unlock()

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		body := `{"tasks":[]}`
		if b, ok := pages[page]; ok {
			body = b
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(body))
	})
	return &urls
}

func anyContains(urls []string, needle string) bool {
	for _, u := range urls {
		if strings.Contains(u, needle) || strings.Contains(u, url.QueryEscape(needle)) {
			return true
		}
	}
	return false
}

// With nothing cached there is nothing to be incremental about: pull the
// workspace, and leave an index behind so the next search does not have to.
func TestSearch_ColdCacheSyncsFullyAndPersists(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir

	urls := recordingMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"A", "iOS 5.6.1 Release Card"}),
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.OutBuf.String(), "iOS 5.6.1 Release Card")
	assert.False(t, anyContains(*urls, "date_updated_gt"), "cold sync must not be incremental")

	idx, err := taskindex.Load(dir, "12345")
	assert.NoError(t, err)
	assert.Contains(t, idx.Entries, "A", "index not persisted")
	assert.False(t, idx.NeedsFullSync(time.Now(), time.Hour), "full sync not recorded")
}

// The point of the cache: a warm index answers from local data and asks the API
// only for what changed. The match here exists solely in the cache — the API
// returns nothing — so finding it proves the search read the index.
func TestSearch_WarmCacheAnswersFromIndexAndFetchesOnlyChanges(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{
		{ID: "cached", Name: "iOS 5.6.1 Release Card", Status: "open", DateUpdated: 1_000_000},
	}, time.Now())
	if err := taskindex.Save(dir, idx); err != nil {
		t.Fatal(err)
	}

	urls := recordingMux(tf, nil) // API has nothing new

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.OutBuf.String(), "iOS 5.6.1 Release Card", "cached match not returned")
	assert.True(t, anyContains(*urls, "date_updated_gt"), "warm search was not incremental")
	assert.Len(t, *urls, 1, "warm search should cost a single request")
}

// Changes since the last sync have to land in the results, and in the cache.
func TestSearch_WarmCacheMergesNewlyChangedTasks(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{
		{ID: "cached", Name: "Old 5.6.1 card", DateUpdated: 1_000_000},
	}, time.Now())
	if err := taskindex.Save(dir, idx); err != nil {
		t.Fatal(err)
	}

	recordingMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"fresh", "Brand new 5.6.1 card"}),
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Old 5.6.1 card")
	assert.Contains(t, out, "Brand new 5.6.1 card")

	reloaded, _ := taskindex.Load(dir, "12345")
	assert.Contains(t, reloaded.Entries, "fresh", "new task not merged into the cache")
}

// A cache old enough to have drifted gets rebuilt, not topped up: incremental
// syncs cannot see deletions.
func TestSearch_StaleCacheTriggersFullResync(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{
		{ID: "gone", Name: "Deleted 5.6.1 card", DateUpdated: 1_000_000},
	}, time.Now().Add(-2*cacheTTL))
	if err := taskindex.Save(dir, idx); err != nil {
		t.Fatal(err)
	}

	urls := recordingMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"live", "Live 5.6.1 card"}),
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.False(t, anyContains(*urls, "date_updated_gt"), "stale cache was topped up, not rebuilt")
	assert.NotContains(t, tf.OutBuf.String(), "Deleted 5.6.1 card", "vanished task survived the rebuild")

	reloaded, _ := taskindex.Load(dir, "12345")
	assert.NotContains(t, reloaded.Entries, "gone")
}

// The cache holds subtasks so it can serve both modes, but a plain search must
// not start showing them.
func TestSearch_CachedSubtasksHiddenUnlessRequested(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{
		{ID: "top", Name: "5.6.1 profiling", DateUpdated: 1_000_000},
		{ID: "sub", Name: "5.6.1 soak test", Parent: "top", DateUpdated: 1_000_000},
	}, time.Now())
	if err := taskindex.Save(dir, idx); err != nil {
		t.Fatal(err)
	}

	recordingMux(tf, nil)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := tf.OutBuf.String()
	assert.Contains(t, out, "5.6.1 profiling")
	assert.NotContains(t, out, "5.6.1 soak test", "subtask shown without --include-subtasks")
}

func TestSearch_CachedSubtasksShownWhenRequested(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{
		{ID: "sub", Name: "5.6.1 soak test", Parent: "top", DateUpdated: 1_000_000},
	}, time.Now())
	if err := taskindex.Save(dir, idx); err != nil {
		t.Fatal(err)
	}

	recordingMux(tf, nil)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--include-subtasks"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.OutBuf.String(), "5.6.1 soak test")
}

// Reading the whole workspace from a local index means the page cap never
// applies, so there is nothing to disclose.
func TestSearch_WarmCacheDoesNotClaimTruncation(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{
		{ID: "a", Name: "5.6.1 card", DateUpdated: 1_000_000},
	}, time.Now())
	if err := taskindex.Save(dir, idx); err != nil {
		t.Fatal(err)
	}

	recordingMux(tf, nil)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.NotContains(t, tf.ErrBuf.String(), "Not shown:")
}

// --no-cache must neither read nor write the index.
func TestSearch_NoCacheFlagBypassesIndexEntirely(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	urls := recordingMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"A", "Live 5.6.1 card"}),
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.False(t, anyContains(*urls, "date_updated_gt"))

	idx, _ := taskindex.Load(dir, "12345")
	assert.Empty(t, idx.Entries, "--no-cache wrote to the index")
}

// Comment bodies are not in the index, so --comments has to go back to the API.
func TestSearch_CommentsFlagBypassesIndex(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)
	recordingMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"A", "Live 5.6.1 card"}),
	})
	tf.Handle("GET", "task/A/comment", 200, `{"comments":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--comments"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	idx, _ := taskindex.Load(dir, "12345")
	assert.Empty(t, idx.Entries, "--comments used the index")
}

// A cache written by an older build, or truncated mid-write, must degrade to a
// cold sync rather than surfacing as an error.
func TestSearch_CorruptCacheFallsBackToFullSync(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	if err := writeFileHelper(dir, "index-12345.json", "{ this is not json"); err != nil {
		t.Fatal(err)
	}

	recordingMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"A", "Recovered 5.6.1 card"}),
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.OutBuf.String(), "Recovered 5.6.1 card")
}

func writeFileHelper(dir, name, content string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)
}

// ---------------------------------------------------------------------------
// Sync failure and truncation
// ---------------------------------------------------------------------------

// The worst outcome available: a cold build that cannot finish, discarded, so
// every later search repeats it and search never works again on a workspace
// large enough to need the index most. Whatever was fetched has to survive.
func TestSearch_PartialColdSyncIsPersisted(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir

	// Page 0 lands, page 1 fails: a stand-in for the deadline or a 5xx.
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
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("a partial sync should degrade, not fail: %v", err)
	}

	idx, _ := taskindex.Load(dir, "12345")
	assert.Contains(t, idx.Entries, "A", "partial sync discarded — next run repeats it")
	assert.Contains(t, tf.OutBuf.String(), "5.6.1 card", "fetched data not searched")
}

// An index that does not cover the workspace must say so. This is the same
// contract the live sweep honours; serving it from a cache changes nothing.
func TestSearch_PartialIndexDisclosesIncompleteness(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	idx := taskindex.New("12345")
	idx.MergePartial([]taskindex.Entry{
		{ID: "a", Name: "5.6.1 card", DateUpdated: 1_000_000},
	}, time.Now())
	if err := taskindex.Save(tf.CacheDir, idx); err != nil {
		t.Fatal(err)
	}
	recordingMux(tf, nil)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.ErrBuf.String(), "Not shown:", "partial index passed off as complete")
}

// A capped incremental sync must not advance the watermark over the range it
// never read, or those tasks are invisible until the weekly rebuild.
func TestSearch_TruncatedIncrementalSyncHoldsTheWatermark(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{{ID: "seed", Name: "Seed", DateUpdated: 1_000_000}}, time.Now())
	if err := taskindex.Save(dir, idx); err != nil {
		t.Fatal(err)
	}

	// Every page full, so the fetch runs to its cap and is truncated.
	full := searchTasksJSON(fillerPairs(100)...)
	pages := map[int]string{}
	for i := 0; i < maxFullSyncPages+1; i++ {
		pages[i] = full
	}
	recordingMux(tf, pages)
	// The query matches nothing, so search now falls through to the space walk.
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "nothingmatchesthis"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reloaded, _ := taskindex.Load(dir, "12345")
	assert.Equal(t, int64(1_000_000), reloaded.SyncedAt,
		"watermark advanced past a range the capped sync never read")
}

// Nothing changed upstream, so nothing should be rewritten. The index is
// multi-megabyte; a needless read-modify-write per search undoes the win.
func TestSearch_WarmNoChangeSyncDoesNotRewriteTheIndex(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	dir := tf.CacheDir
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{{ID: "a", Name: "5.6.1 card", DateUpdated: 1_000_000}}, time.Now())
	if err := taskindex.Save(dir, idx); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(filepath.Join(dir, "index-12345.json"))
	if err != nil {
		t.Fatal(err)
	}
	recordingMux(tf, nil) // API reports nothing changed

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after, err := os.Stat(filepath.Join(dir, "index-12345.json"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, before.ModTime(), after.ModTime(), "index rewritten with no changes")
}

// A complete index covers the same tasks the space walk would visit, so "no
// match" is authoritative and walking every list in the workspace can only
// spend minutes to confirm it. Say so instead.
func TestSearch_CompleteIndexWithNoMatchDoesNotWalkTheTree(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{{ID: "a", Name: "Something else", DateUpdated: 1_000_000}}, time.Now())
	if err := taskindex.Save(tf.CacheDir, idx); err != nil {
		t.Fatal(err)
	}
	recordingMux(tf, nil)

	var walked bool
	tf.HandleFunc("team/12345/space", func(w http.ResponseWriter, r *http.Request) {
		walked = true
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"spaces":[]}`))
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.False(t, walked, "walked the whole tree to confirm a complete index")
}

// An incomplete index used to trigger an automatic full-tree walk. Measured on
// the real workspace that was 234 requests and 112 seconds per zero-result
// search — multiplied by word count once the per-word retry joined in — which
// reinstated exactly the cost the index exists to remove. The index now says it
// is incomplete and leaves the walk to the user.
func TestSearch_PartialIndexWithNoMatchDisclosesInsteadOfWalking(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	idx := taskindex.New("12345")
	idx.MergePartial([]taskindex.Entry{
		{ID: "a", Name: "Something else", DateUpdated: 1_000_000},
	}, time.Now())
	if err := taskindex.Save(tf.CacheDir, idx); err != nil {
		t.Fatal(err)
	}
	recordingMux(tf, nil)

	var walked bool
	tf.HandleFunc("team/12345/space", func(w http.ResponseWriter, r *http.Request) {
		walked = true
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"spaces":[]}`))
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.False(t, walked, "walked the whole tree as a reflex on a partial index")
	assert.Contains(t, tf.ErrBuf.String(), "Not shown:", "incompleteness not disclosed")
}

// A weekly rebuild is not a first run, and telling the user it is makes a
// multi-minute wait look like something has gone wrong.
func TestSearch_RebuildIsNotAnnouncedAsAFirstRun(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	idx := taskindex.New("12345")
	idx.Replace([]taskindex.Entry{{ID: "a", Name: "5.6.1 card", DateUpdated: 1_000}}, time.Now().Add(-2*cacheTTL))
	if err := taskindex.Save(tf.CacheDir, idx); err != nil {
		t.Fatal(err)
	}
	recordingMux(tf, map[int]string{0: searchTasksJSON([2]string{"a", "5.6.1 card"})})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.NotContains(t, tf.ErrBuf.String(), "first run")
}
