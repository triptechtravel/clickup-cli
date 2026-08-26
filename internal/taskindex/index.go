// Package taskindex keeps a local mirror of a workspace's tasks so search can
// filter locally instead of paginating the API on every invocation.
//
// This exists because ClickUp has no server-side text search: GET
// team/{id}/task accepts a `search=` param and ignores it, so every match has
// to be found by pulling full pages and filtering them client-side. Doing that
// live puts the two things a user wants — a cheap search and a complete one —
// in direct opposition, since the only lever is how many pages to pull.
//
// An index breaks the tie. The corpus is fetched once, then kept current with
// `date_updated_gt`, which returns only what changed (a week's worth is a
// single page on a busy workspace). Search then reads every task the workspace
// has, at the cost of roughly one request.
package taskindex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// overlapMillis is how far Since() reaches back before the watermark.
//
// The watermark is the newest date_updated we have seen. Asking for strictly
// newer than that would drop a task updated in the same millisecond as the
// last row of a page but written after the page was cut — it would never again
// be newer than the watermark, so it would stay invisible forever. A minute of
// overlap costs a handful of redundant rows and closes the hole.
const overlapMillis int64 = 60_000

// Entry is the searchable projection of a task. Descriptions are kept in full
// because search matches against them.
type Entry struct {
	ID          string `json:"id"`
	CustomID    string `json:"custom_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	// Priority is kept because search renders it in --json. Dropping it would
	// make the same query emit different JSON depending on whether the cache
	// happened to be warm.
	Priority string `json:"priority,omitempty"`
	// Parent is the parent task ID, empty for top-level tasks. The sync always
	// pulls subtasks so one cache serves both --include-subtasks and plain
	// searches; this is what lets the caller tell them apart.
	Parent      string   `json:"parent,omitempty"`
	Assignees   []string `json:"assignees,omitempty"`
	URL         string   `json:"url,omitempty"`
	DateUpdated int64    `json:"date_updated"`
}

// equal reports whether two entries carry the same data. Written out because
// Entry holds a slice and so is not comparable with ==.
func (e Entry) equal(o Entry) bool {
	if e.ID != o.ID || e.CustomID != o.CustomID || e.Name != o.Name ||
		e.Description != o.Description || e.Status != o.Status ||
		e.Priority != o.Priority || e.Parent != o.Parent || e.URL != o.URL ||
		e.DateUpdated != o.DateUpdated || len(e.Assignees) != len(o.Assignees) {
		return false
	}
	for i := range e.Assignees {
		if e.Assignees[i] != o.Assignees[i] {
			return false
		}
	}
	return true
}

// Index is a workspace's task mirror plus the bookkeeping needed to refresh it.
type Index struct {
	Workspace string `json:"workspace"`
	// SyncedAt is the newest date_updated seen, in epoch millis.
	SyncedAt int64 `json:"synced_at"`
	// FullSyncAt is when the index was last rebuilt from scratch, in epoch
	// millis. Incremental syncs cannot see deletions, so only a rebuild can
	// drop tasks that have gone away.
	FullSyncAt int64 `json:"full_sync_at"`
	// Partial records that the last rebuild hit its page cap and so does not
	// cover the whole workspace. Searches served from it must disclose that
	// rather than presenting an empty result as authoritative.
	Partial bool             `json:"partial,omitempty"`
	Entries map[string]Entry `json:"entries"`
}

// New returns an empty index for a workspace.
func New(workspace string) *Index {
	return &Index{Workspace: workspace, Entries: map[string]Entry{}}
}

// Merge upserts entries by ID and advances the watermark, reporting whether
// anything actually changed. It never removes anything, and never moves the
// watermark backwards.
func (i *Index) Merge(entries []Entry) bool {
	changed := i.upsert(entries)
	for _, e := range entries {
		if e.DateUpdated > i.SyncedAt {
			i.SyncedAt = e.DateUpdated
			changed = true
		}
	}
	return changed
}

// MergeKeepingWatermark upserts entries without advancing the watermark.
//
// For a fetch that was cut short. The fetch reads newest-first, so a capped run
// covers [floor, newest] and leaves [watermark, floor) unread. Advancing to
// `newest` would strand that gap for good — everything in it is older than the
// new watermark, so no later incremental sync would ask for it again. Standing
// still costs a repeated read; moving costs the tasks.
func (i *Index) MergeKeepingWatermark(entries []Entry) bool {
	return i.upsert(entries)
}

func (i *Index) upsert(entries []Entry) bool {
	if i.Entries == nil {
		i.Entries = map[string]Entry{}
	}
	changed := false
	for _, e := range entries {
		if existing, ok := i.Entries[e.ID]; ok && existing.equal(e) {
			continue
		}
		i.Entries[e.ID] = e
		changed = true
	}
	return changed
}

// Replace rebuilds the index from a complete fetch, dropping anything absent
// from it, and records the reconcile time.
func (i *Index) Replace(entries []Entry, at time.Time) {
	i.Entries = make(map[string]Entry, len(entries))
	i.SyncedAt = 0
	i.Merge(entries)
	i.FullSyncAt = at.UnixMilli()
	i.Partial = false
}

// MergePartial records a rebuild that hit its page cap. It merges rather than
// replaces — a fetch that did not see the whole workspace cannot be used to
// decide what has been deleted from it — and flags the index incomplete.
//
// The reconcile clock is still stamped: without that, an index too large to
// rebuild in one pass would retry the same oversized sync on every search.
func (i *Index) MergePartial(entries []Entry, at time.Time) bool {
	changed := i.Merge(entries)
	i.FullSyncAt = at.UnixMilli()
	if !i.Partial {
		i.Partial = true
		changed = true
	}
	return changed
}

// Since returns the date_updated_gt value for the next incremental sync, or 0
// if the index has never been populated.
func (i *Index) Since() int64 {
	if i.SyncedAt == 0 {
		return 0
	}
	if i.SyncedAt < overlapMillis {
		return 0
	}
	return i.SyncedAt - overlapMillis
}

// NeedsFullSync reports whether the index should be rebuilt rather than topped
// up — either it has never been built, or it is old enough that accumulated
// deletions are worth reconciling.
func (i *Index) NeedsFullSync(now time.Time, ttl time.Duration) bool {
	if i.FullSyncAt == 0 {
		return true
	}
	return now.Sub(time.UnixMilli(i.FullSyncAt)) > ttl
}

// All returns every indexed entry.
func (i *Index) All() []Entry {
	out := make([]Entry, 0, len(i.Entries))
	for _, e := range i.Entries {
		out = append(out, e)
	}
	return out
}

func path(dir, workspace string) string {
	return filepath.Join(dir, fmt.Sprintf("index-%s.json", workspace))
}

// Load reads a workspace's index. A missing or unreadable cache is not an
// error: it yields an empty index, so the caller does a cold sync rather than
// failing. A corrupt cache is a performance problem, never a correctness one.
func Load(dir, workspace string) (*Index, error) {
	b, err := os.ReadFile(path(dir, workspace))
	if err != nil {
		return New(workspace), nil
	}

	var idx Index
	if err := json.Unmarshal(b, &idx); err != nil {
		return New(workspace), nil
	}
	if idx.Entries == nil {
		idx.Entries = map[string]Entry{}
	}
	idx.Workspace = workspace
	return &idx, nil
}

// Save writes the index, creating the cache directory if needed. The file
// holds task names and descriptions, so it is owner-only.
func Save(dir string, idx *Index) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	b, err := json.Marshal(idx)
	if err != nil {
		return fmt.Errorf("encode index: %w", err)
	}
	return os.WriteFile(path(dir, idx.Workspace), b, 0o600)
}
