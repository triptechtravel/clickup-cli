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
	"regexp"
	"strings"
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

// maxDescriptionBytes bounds what is kept of a task description.
//
// Descriptions are matched against, so they have to be stored — but they are
// also two thirds of the index by size, with a median of 97 bytes and a tail
// running to 35KB. Bounding the tail halves the file for a recall loss confined
// to text buried deep inside unusually long bodies.
const maxDescriptionBytes = 4096

// MaxDescriptionBytes exposes the description bound so the command can state it.
const MaxDescriptionBytes = maxDescriptionBytes

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
	// IndexedAt is when this entry was last fetched, in epoch millis. It is
	// what makes a multi-segment rebuild able to reconcile deletions: anything
	// still carrying a timestamp from before the rebuild began was never seen
	// during it, and so is gone upstream.
	IndexedAt int64 `json:"indexed_at,omitempty"`
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
	Partial bool `json:"partial,omitempty"`
	// RebuildFloor is the oldest date_updated a partial rebuild reached, in
	// epoch millis, or 0 when no rebuild is in progress. The next segment
	// continues below it, so a workspace too large to rebuild inside one sync
	// budget converges over consecutive searches instead of failing forever.
	RebuildFloor int64 `json:"rebuild_floor,omitempty"`
	// RebuildStartedAt is when the current rebuild began, in epoch millis.
	RebuildStartedAt int64            `json:"rebuild_started_at,omitempty"`
	Entries          map[string]Entry `json:"entries"`
}

// New returns an empty index for a workspace.
// validWorkspace guards the one piece of caller-controlled text that becomes a
// filename. It arrives from the ClickUp API and is normally numeric, but
// nothing checked, so a value containing ../ escaped the cache directory
// entirely and overwrote whatever .json file it landed on.
var validWorkspace = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func checkWorkspace(workspace string) error {
	if !validWorkspace.MatchString(workspace) {
		return fmt.Errorf("refusing to use %q as a workspace id: it is not a plain identifier and would name a file outside the cache", workspace)
	}
	return nil
}

// sanitizeText removes control characters from text that came off disk.
//
// The renderer writes names straight to a terminal, and the cache is a local
// file: anything able to write it could replay escape sequences into the user's
// terminal on every search, or dress up a task id an agent then acts on.
func sanitizeText(in string) string {
	if !strings.ContainsFunc(in, isControl) {
		return in
	}
	return strings.Map(func(r rune) rune {
		if isControl(r) {
			return -1
		}
		return r
	}, in)
}

func isControl(r rune) bool {
	return r == 0x7f || (r < 0x20 && r != '\t' && r != '\n')
}

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
		if len(e.Description) > maxDescriptionBytes {
			e.Description = e.Description[:maxDescriptionBytes]
		}
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
	i.Merge(entries)
	i.FullSyncAt = at.UnixMilli()
	i.Partial = true
	// Always a change: FullSyncAt moved. Reporting otherwise let the caller
	// skip the write, leaving the reconcile clock stale on disk — so the next
	// search redid the same oversized rebuild, and so did the one after that.
	return true
}

// RequestFullSync clears the reconcile clock so the next search rebuilds.
//
// For a fetch that was cut short and cannot catch up on its own: an
// incremental sync holds its watermark when truncated, which is what keeps the
// unread gap from being stranded, but it also means repeating the same capped
// read forever. A rebuild is the way out.
func (i *Index) RequestFullSync() {
	i.FullSyncAt = 0
}

// Since returns the date_updated_gt value for the next incremental sync, or 0
// if the index has never been populated.
func (i *Index) Since() int64 {
	return i.SinceAt(time.Now())
}

// SinceAt is Since as of a given moment, clamped so a timestamp in the future
// cannot poison the watermark.
//
// date_updated is server-set, but importers and integrations do emit rows dated
// ahead of now. One such row pushed the watermark past every real change, so
// every later incremental asked for something newer than the future and got
// nothing — silently indexing no new work until the weekly rebuild re-ingested
// the same row and did it again.
func (i *Index) SinceAt(now time.Time) int64 {
	watermark := i.SyncedAt
	if ms := now.UnixMilli(); watermark > ms {
		watermark = ms
	}
	if watermark < overlapMillis {
		return 0
	}
	return watermark - overlapMillis
}

// BeginRebuild marks the start of a full rebuild pass.
func (i *Index) BeginRebuild(at time.Time) {
	i.RebuildStartedAt = at.UnixMilli()
	i.RebuildFloor = 0
}

// AdvanceRebuild records one completed segment of a rebuild that has not
// finished, so the next run can pick up where this one stopped.
//
// It deliberately leaves SyncedAt alone: a rebuild in flight has not
// established a new incremental watermark, and moving it would strand every
// change between the old watermark and this segment's floor.
func (i *Index) AdvanceRebuild(entries []Entry, floor int64) bool {
	i.upsert(entries)
	if floor != 0 {
		i.RebuildFloor = floor
	} else if i.RebuildFloor == 0 {
		// Nothing came back and no earlier segment set a floor. Leave a marker
		// so NeedsFullSync keeps the rebuild alive rather than treating this as
		// a finished pass.
		i.RebuildFloor = -1
	}
	i.Partial = true
	return true
}

// CompleteRebuild finishes a rebuild: it merges the final segment, drops every
// task the rebuild never saw, and restores the index to complete.
//
// It refuses, returning false, when a rebuild that saw nothing at all is asked
// to reconcile a populated index. An empty response is far more likely to be a
// bad page than a workspace that lost every task, and accepting it wipes the
// mirror and then answers "no tasks found" with total confidence.
func (i *Index) CompleteRebuild(entries []Entry, at time.Time) bool {
	i.upsert(entries)

	started := i.RebuildStartedAt
	seen := 0
	for _, e := range i.Entries {
		if e.IndexedAt >= started && started > 0 {
			seen++
		}
	}
	if seen == 0 && len(i.Entries) > 0 {
		i.Partial = true
		i.RebuildFloor = 0
		return false
	}

	if started > 0 {
		for id, e := range i.Entries {
			if e.IndexedAt < started {
				delete(i.Entries, id)
			}
		}
	}

	// The rebuild walked the whole corpus, so the newest row in it is the
	// watermark every later incremental starts from.
	i.SyncedAt = 0
	for _, e := range i.Entries {
		if e.DateUpdated > i.SyncedAt {
			i.SyncedAt = e.DateUpdated
		}
	}

	i.FullSyncAt = at.UnixMilli()
	i.RebuildFloor = 0
	i.RebuildStartedAt = 0
	i.Partial = false
	return true
}

// NeedsFullSync reports whether the index should be rebuilt rather than topped
// up — either it has never been built, or it is old enough that accumulated
// deletions are worth reconciling.
func (i *Index) NeedsFullSync(now time.Time, ttl time.Duration) bool {
	if i.RebuildFloor != 0 {
		// A rebuild is mid-flight; finishing it takes priority over any clock.
		return true
	}
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

// reapTemps removes abandoned temp files.
//
// The write-then-rename cleanup is deferred, which covers error returns but not
// signal death — and the first build is advertised as slow, so Ctrl-C during it
// is the expected reaction. Each interrupt left a temp file holding every task
// name and description fetched so far, and nothing ever removed them.
func reapTemps(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), ".index-") || !strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < time.Hour {
			continue
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

// Load reads a workspace's index. A missing or unreadable cache is not an
// error: it yields an empty index, so the caller does a cold sync rather than
// failing. A corrupt cache is a performance problem, never a correctness one.
func Load(dir, workspace string) (*Index, error) {
	if err := checkWorkspace(workspace); err != nil {
		return nil, err
	}
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
	for id, e := range idx.Entries {
		e.Name = sanitizeText(e.Name)
		e.Description = sanitizeText(e.Description)
		e.Status = sanitizeText(e.Status)
		idx.Entries[id] = e
	}
	idx.Workspace = workspace
	return &idx, nil
}

// Save writes the index, creating the cache directory if needed. The file
// holds task names and descriptions, so it is owner-only.
func Save(dir string, idx *Index) error {
	if err := checkWorkspace(idx.Workspace); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	// MkdirAll leaves an existing directory's mode alone, and this one holds
	// the whole workspace in plaintext.
	_ = os.Chmod(dir, 0o700)
	reapTemps(dir)
	b, err := json.Marshal(idx)
	if err != nil {
		return fmt.Errorf("encode index: %w", err)
	}

	// Write-then-rename. os.WriteFile truncates in place, so a reader racing a
	// writer — two searches at once, which is routine when agents drive this
	// CLI — sees half a file. Load treats that as an empty index, so the
	// symptom is a surprise multi-minute rebuild rather than an error.
	final := path(dir, idx.Workspace)
	tmp, err := os.CreateTemp(dir, ".index-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp index: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("write index: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod index: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close index: %w", err)
	}
	return os.Rename(tmpName, final)
}
