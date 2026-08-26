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
