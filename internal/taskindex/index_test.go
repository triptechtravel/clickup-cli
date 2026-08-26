package taskindex

import (
	"os"
	"path/filepath"
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
