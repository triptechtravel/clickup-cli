package taskindex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func entry(id, name string, updated int64) Entry {
	return Entry{ID: id, Name: name, DateUpdated: updated}
}

// ---------------------------------------------------------------------------
// Merge
// ---------------------------------------------------------------------------

func TestIndex_MergeUpsertsByID(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{entry("a", "Old name", 100)})
	idx.Merge([]Entry{entry("a", "New name", 200)})

	assert.Len(t, idx.Entries, 1)
	assert.Equal(t, "New name", idx.Entries["a"].Name)
}

func TestIndex_MergeAdvancesWatermark(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{entry("a", "A", 100), entry("b", "B", 500)})

	assert.Equal(t, int64(500), idx.SyncedAt)
}

// A late-arriving old task must not drag the watermark backwards, or the next
// incremental sync re-downloads everything since that date.
func TestIndex_MergeDoesNotRewindWatermark(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{entry("a", "A", 500)})
	idx.Merge([]Entry{entry("b", "B", 100)})

	assert.Equal(t, int64(500), idx.SyncedAt)
}

// Since() deliberately reaches back before the watermark. A task updated in the
// same millisecond as the last one we saw, but after the page was cut, would
// otherwise fall in the gap and never be picked up again.
func TestIndex_SinceOverlapsTheWatermark(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{entry("a", "A", 1_000_000_000)})

	assert.Less(t, idx.Since(), idx.SyncedAt, "no overlap window")
	assert.Equal(t, idx.SyncedAt-overlapMillis, idx.Since())
}

func TestIndex_SinceIsZeroWhenNeverSynced(t *testing.T) {
	assert.Equal(t, int64(0), New("12345").Since())
}

// ---------------------------------------------------------------------------
// Replace (full reconcile)
// ---------------------------------------------------------------------------

// Incremental syncs only ever add. Tasks deleted or archived upstream are
// invisible to them, so a full sync has to drop what it no longer sees.
func TestIndex_ReplaceDropsVanishedTasks(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{entry("a", "A", 100), entry("gone", "Deleted", 100)})

	now := time.UnixMilli(9_000_000)
	idx.Replace([]Entry{entry("a", "A", 100)}, now)

	assert.Len(t, idx.Entries, 1)
	assert.NotContains(t, idx.Entries, "gone")
	assert.Equal(t, now.UnixMilli(), idx.FullSyncAt)
}

// ---------------------------------------------------------------------------
// Staleness
// ---------------------------------------------------------------------------

func TestIndex_NeedsFullSyncWhenNeverSynced(t *testing.T) {
	assert.True(t, New("12345").NeedsFullSync(time.Now(), 7*24*time.Hour))
}

func TestIndex_NeedsFullSyncWhenStale(t *testing.T) {
	idx := New("12345")
	now := time.UnixMilli(1_000_000_000)
	idx.Replace(nil, now.Add(-8*24*time.Hour))

	assert.True(t, idx.NeedsFullSync(now, 7*24*time.Hour))
}

func TestIndex_NoFullSyncWhenFresh(t *testing.T) {
	idx := New("12345")
	now := time.UnixMilli(1_000_000_000)
	idx.Replace(nil, now.Add(-1*time.Hour))

	assert.False(t, idx.NeedsFullSync(now, 7*24*time.Hour))
}

// ---------------------------------------------------------------------------
// Persistence
// ---------------------------------------------------------------------------

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	idx := New("12345")
	idx.Merge([]Entry{
		{ID: "a", CustomID: "CU-a", Name: "A task", Description: "body", Status: "open",
			Assignees: []string{"isaac"}, URL: "https://x", DateUpdated: 100},
	})

	if err := Save(dir, idx); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := Load(dir, "12345")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	assert.Equal(t, idx.SyncedAt, got.SyncedAt)
	assert.Equal(t, idx.Entries["a"], got.Entries["a"])
}

func TestLoad_MissingFileReturnsEmptyIndex(t *testing.T) {
	got, err := Load(t.TempDir(), "12345")

	assert.NoError(t, err)
	assert.Empty(t, got.Entries)
	assert.True(t, got.NeedsFullSync(time.Now(), time.Hour))
}

// A corrupt cache is a performance problem, never a correctness one: search
// must fall back to a cold sync rather than failing in the user's face.
func TestLoad_CorruptFileReturnsEmptyIndexNotError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index-12345.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(dir, "12345")

	assert.NoError(t, err)
	assert.Empty(t, got.Entries)
}

// Two workspaces must not share a cache file.
func TestSaveLoad_IsPerWorkspace(t *testing.T) {
	dir := t.TempDir()
	a := New("aaa")
	a.Merge([]Entry{entry("1", "In A", 100)})
	b := New("bbb")
	b.Merge([]Entry{entry("2", "In B", 100)})

	assert.NoError(t, Save(dir, a))
	assert.NoError(t, Save(dir, b))

	gotA, _ := Load(dir, "aaa")
	assert.Contains(t, gotA.Entries, "1")
	assert.NotContains(t, gotA.Entries, "2")
}

// The cache holds task names and descriptions; it should not be world-readable.
func TestSave_FileIsNotWorldReadable(t *testing.T) {
	dir := t.TempDir()
	assert.NoError(t, Save(dir, New("12345")))

	info, err := os.Stat(filepath.Join(dir, "index-12345.json"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

// The sync always pulls subtasks so one cache can serve both --include-subtasks
// and plain searches; the entry has to remember which is which.
func TestIndex_RemembersParent(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{{ID: "child", Name: "Soak test", Parent: "parent", DateUpdated: 100}})

	assert.Equal(t, "parent", idx.Entries["child"].Parent)
}

// ---------------------------------------------------------------------------
// Truncated syncs
// ---------------------------------------------------------------------------

// A capped incremental fetch reads newest-first down to some floor, leaving a
// gap between the old watermark and that floor. Advancing the watermark past
// the gap would make those tasks permanently invisible: they are older than
// the new watermark, so no later incremental sync would ever ask for them.
// Re-reading is cheap; losing them is not.
func TestIndex_MergeKeepingWatermarkDoesNotAdvancePastAGap(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{entry("old", "Old", 100)})

	idx.MergeKeepingWatermark([]Entry{entry("new", "New", 9_000)})

	assert.Contains(t, idx.Entries, "new", "entries still merge")
	assert.Equal(t, int64(100), idx.SyncedAt, "watermark advanced over an unread gap")
}

// A capped full sync has not seen the whole workspace, so it must not be used
// to decide what no longer exists.
func TestIndex_MarkPartialKeepsUnseenEntries(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{entry("older", "Older task", 100)})

	now := time.UnixMilli(9_000_000)
	idx.MergePartial([]Entry{entry("newer", "Newer task", 500)}, now)

	assert.Contains(t, idx.Entries, "older", "a capped rebuild deleted what it never read")
	assert.Contains(t, idx.Entries, "newer")
	assert.True(t, idx.Partial, "index not flagged incomplete")
}

// The reconcile clock still ticks on a partial rebuild, otherwise every search
// retries the same oversized sync.
func TestIndex_MarkPartialStillRecordsTheAttempt(t *testing.T) {
	idx := New("12345")
	now := time.UnixMilli(9_000_000)
	idx.MergePartial(nil, now)

	assert.False(t, idx.NeedsFullSync(now, time.Hour), "partial rebuild retries immediately")
}

// A complete rebuild clears the incomplete flag.
func TestIndex_ReplaceClearsPartial(t *testing.T) {
	idx := New("12345")
	idx.MergePartial(nil, time.UnixMilli(1))
	idx.Replace([]Entry{entry("a", "A", 100)}, time.UnixMilli(2))

	assert.False(t, idx.Partial)
}

// ---------------------------------------------------------------------------
// Change tracking
// ---------------------------------------------------------------------------

// The common case is a search that changes nothing. Rewriting a multi-megabyte
// index for that undercuts the point of having one.
func TestIndex_MergeReportsWhetherAnythingChanged(t *testing.T) {
	idx := New("12345")
	assert.True(t, idx.Merge([]Entry{entry("a", "A", 100)}), "first insert is a change")
	assert.False(t, idx.Merge([]Entry{entry("a", "A", 100)}), "identical re-merge is not a change")
	assert.True(t, idx.Merge([]Entry{entry("a", "A renamed", 200)}), "updated task is a change")
	assert.False(t, idx.Merge(nil), "empty merge is not a change")
}

// ---------------------------------------------------------------------------
// Projection completeness
// ---------------------------------------------------------------------------

// Priority is rendered in --json output. If the index drops it, the same query
// returns different JSON depending on whether the cache was warm.
func TestIndex_KeepsPriority(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{{ID: "a", Name: "A", Priority: "high", DateUpdated: 100}})

	assert.Equal(t, "high", idx.Entries["a"].Priority)
}

// A truncated sync must leave a way to catch up, or every later search repeats
// the same capped read.
func TestIndex_RequestFullSyncForcesARebuild(t *testing.T) {
	idx := New("12345")
	idx.Replace(nil, time.Now())
	assert.False(t, idx.NeedsFullSync(time.Now(), time.Hour))

	idx.RequestFullSync()

	assert.True(t, idx.NeedsFullSync(time.Now(), time.Hour))
}

// The reconcile clock moves on every partial rebuild, so the caller must be
// told to persist even when no entry changed.
func TestIndex_MergePartialAlwaysReportsAChange(t *testing.T) {
	idx := New("12345")
	idx.MergePartial(nil, time.UnixMilli(1))

	assert.True(t, idx.MergePartial(nil, time.UnixMilli(2)), "clock moved but no write requested")
}

// A half-written index must never be observable: readers see the old file or
// the new one, nothing in between.
func TestSave_LeavesNoPartialFileBehind(t *testing.T) {
	dir := t.TempDir()
	idx := New("12345")
	idx.Merge([]Entry{entry("a", "A", 100)})
	if err := Save(dir, idx); err != nil {
		t.Fatal(err)
	}
	idx.Merge([]Entry{entry("b", "B", 200)})
	if err := Save(dir, idx); err != nil {
		t.Fatal(err)
	}

	got, err := Load(dir, "12345")
	assert.NoError(t, err)
	assert.Len(t, got.Entries, 2)

	files, _ := os.ReadDir(dir)
	for _, f := range files {
		assert.False(t, strings.HasPrefix(f.Name(), ".index-"), "temp file left behind: %s", f.Name())
	}
}

// ---------------------------------------------------------------------------
// Resumable rebuild
//
// A rebuild that cannot finish inside one sync budget used to give up and mark
// the index permanently incomplete: past roughly 10,500 tasks every rebuild hit
// the wall, flagged Partial, stamped the reconcile clock, and the next one hit
// the same wall a week later. Rebuilds now carry a floor cursor and resume from
// it, so a large workspace converges over consecutive searches.
// ---------------------------------------------------------------------------

func TestIndex_TruncatedRebuildRecordsAResumePoint(t *testing.T) {
	idx := New("12345")
	now := time.UnixMilli(9_000_000)
	idx.BeginRebuild(now)

	idx.AdvanceRebuild([]Entry{entry("a", "A", 900), entry("b", "B", 500)}, 500)

	assert.Equal(t, int64(500), idx.RebuildFloor, "no resume point recorded")
	assert.True(t, idx.Partial)
	assert.Contains(t, idx.Entries, "a")
}

// The resume point is where the next segment starts, and it must not be
// mistaken for the incremental watermark.
func TestIndex_TruncatedRebuildDoesNotTouchTheWatermark(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{entry("seed", "Seed", 1000)})
	before := idx.SyncedAt

	idx.BeginRebuild(time.UnixMilli(9_000_000))
	idx.AdvanceRebuild([]Entry{entry("a", "A", 500_000)}, 400_000)

	assert.Equal(t, before, idx.SyncedAt, "a mid-flight rebuild moved the incremental watermark")
}

// Finishing the rebuild is what makes the index complete again.
func TestIndex_CompletedRebuildClearsPartialAndResumePoint(t *testing.T) {
	idx := New("12345")
	now := time.UnixMilli(9_000_000)
	idx.BeginRebuild(now)
	idx.AdvanceRebuild([]Entry{{ID: "a", Name: "A", DateUpdated: 900, IndexedAt: now.UnixMilli()}}, 900)

	idx.CompleteRebuild([]Entry{{ID: "b", Name: "B", DateUpdated: 100, IndexedAt: now.UnixMilli()}}, now)

	assert.False(t, idx.Partial)
	assert.Zero(t, idx.RebuildFloor)
	assert.False(t, idx.NeedsFullSync(now, time.Hour))
	assert.Equal(t, int64(900), idx.SyncedAt, "watermark not set from the completed rebuild")
	assert.Contains(t, idx.Entries, "a", "an earlier segment's entries were dropped")
	assert.Contains(t, idx.Entries, "b")
}

// Deletions are reconciled across the whole multi-segment rebuild, not just the
// final segment: anything not seen during the rebuild is gone upstream.
func TestIndex_CompletedRebuildDropsTasksNotSeenInAnySegment(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{{ID: "stale", Name: "Deleted upstream", DateUpdated: 50, IndexedAt: 1}})

	now := time.UnixMilli(9_000_000)
	idx.BeginRebuild(now)
	idx.AdvanceRebuild([]Entry{{ID: "kept", Name: "Kept", DateUpdated: 900, IndexedAt: now.UnixMilli()}}, 900)
	idx.CompleteRebuild(nil, now)

	assert.Contains(t, idx.Entries, "kept")
	assert.NotContains(t, idx.Entries, "stale", "deleted task survived a completed rebuild")
}

// A rebuild that comes back empty against a populated index is far more likely
// to be a bad response than a workspace that lost every task, and treating it
// as truth wipes the mirror and answers "no tasks found" with confidence.
func TestIndex_CompletedRebuildRefusesToEmptyAPopulatedIndex(t *testing.T) {
	idx := New("12345")
	idx.Merge([]Entry{entry("a", "A", 100), entry("b", "B", 200)})

	now := time.UnixMilli(9_000_000)
	idx.BeginRebuild(now)
	ok := idx.CompleteRebuild(nil, now)

	assert.False(t, ok, "an empty rebuild was accepted as authoritative")
	assert.Len(t, idx.Entries, 2, "index wiped by a single empty response")
	assert.True(t, idx.Partial, "wiped index not flagged for another attempt")
}

// A rebuild in progress must continue, whatever the reconcile clock says.
func TestIndex_RebuildInProgressForcesContinuation(t *testing.T) {
	idx := New("12345")
	now := time.UnixMilli(9_000_000)
	idx.BeginRebuild(now)
	idx.AdvanceRebuild([]Entry{entry("a", "A", 900)}, 900)

	assert.True(t, idx.NeedsFullSync(now, time.Hour), "resume point ignored")
}

// ---------------------------------------------------------------------------
// Watermark sanity
// ---------------------------------------------------------------------------

// date_updated is server-set, but importers and integrations do produce rows
// dated in the future. One such row set the watermark ahead of now, and every
// later incremental asked for changes newer than that — returning nothing,
// forever, with no disclosure.
func TestIndex_SinceIgnoresAFutureWatermark(t *testing.T) {
	idx := New("12345")
	now := time.UnixMilli(1_000_000_000)
	idx.Merge([]Entry{entry("skewed", "Imported", now.UnixMilli()+31_536_000_000)})

	assert.LessOrEqual(t, idx.SinceAt(now), now.UnixMilli(),
		"a future timestamp poisoned the incremental watermark")
}

// ---------------------------------------------------------------------------
// Concurrent processes
// ---------------------------------------------------------------------------

// Two searches at once each loaded, synced and saved, so the last writer threw
// away the other's work entirely — and because a truncated sync stamps the
// reconcile clock, the loser's complete rebuild could be replaced by the
// winner's partial one and pinned there for a week.
func TestUpdate_ConcurrentWritersDoNotLoseEachOthersEntries(t *testing.T) {
	dir := t.TempDir()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = Update(dir, "12345", func(idx *Index) bool {
				return idx.Merge([]Entry{entry(fmt.Sprintf("t%d", n), "Task", int64(100+n))})
			})
		}(i)
	}
	wg.Wait()

	got, err := Load(dir, "12345")
	assert.NoError(t, err)
	assert.Len(t, got.Entries, 8, "concurrent writers lost entries")
}

// Update must read the state it is about to modify, not one captured earlier.
func TestUpdate_AppliesToTheCurrentOnDiskState(t *testing.T) {
	dir := t.TempDir()
	seed := New("12345")
	seed.Merge([]Entry{entry("existing", "Existing", 100)})
	if err := Save(dir, seed); err != nil {
		t.Fatal(err)
	}

	err := Update(dir, "12345", func(idx *Index) bool {
		assert.Contains(t, idx.Entries, "existing", "Update did not see what was on disk")
		return idx.Merge([]Entry{entry("added", "Added", 200)})
	})
	assert.NoError(t, err)

	got, _ := Load(dir, "12345")
	assert.Len(t, got.Entries, 2)
}
