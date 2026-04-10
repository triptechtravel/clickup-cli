package inbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/raksul/go-clickup/clickup"
	"github.com/triptechtravel/clickup-cli/internal/config"
)

const (
	inboxCacheFilename = "inbox_cache.json"
	inboxCacheTTL      = 24 * time.Hour
)

const (
	eventAssignment         = "assignment"
	eventDescriptionMention = "description_mention"
	eventCommentMention     = "comment_mention"
)

// inboxCache stores the date_updated of tasks observed in the workspace scan
// so subsequent runs can skip the comment fetch for tasks that have not changed.
type inboxCache struct {
	ScannedAt int64                      `json:"scanned_at"`
	Tasks     map[string]inboxCacheEntry `json:"tasks"`
}

type inboxCacheEntry struct {
	DateUpdated string    `json:"date_updated"`
	Mentions    []mention `json:"mentions,omitempty"`
}

func inboxCachePath() string {
	return filepath.Join(config.ConfigDir(), inboxCacheFilename)
}

func loadInboxCache(path string) (*inboxCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &inboxCache{Tasks: map[string]inboxCacheEntry{}}, nil
		}
		return nil, err
	}

	var c inboxCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	if c.Tasks == nil {
		c.Tasks = map[string]inboxCacheEntry{}
	}
	return &c, nil
}

func saveInboxCache(path string, c *inboxCache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// IsFresh reports whether the cache was scanned within the TTL window.
func (c *inboxCache) IsFresh(now time.Time) bool {
	if c == nil || c.ScannedAt == 0 {
		return false
	}
	return now.Sub(time.UnixMilli(c.ScannedAt)) < inboxCacheTTL
}

// cacheDiff returns the subset of tasks whose date_updated has changed since the
// cache was last written. A nil or empty cache returns all tasks (cold start).
func cacheDiff(cache *inboxCache, tasks []clickup.Task) []clickup.Task {
	if cache == nil || len(cache.Tasks) == 0 {
		return tasks
	}
	var changed []clickup.Task
	for _, t := range tasks {
		entry, ok := cache.Tasks[t.ID]
		if !ok || entry.DateUpdated != t.DateUpdated {
			changed = append(changed, t)
		}
	}
	return changed
}

// updateCacheFromTasks records the latest date_updated for each task and stamps
// the cache with the current scan time. The mentionsByTask map associates the
// mentions discovered in this run with their parent task so subsequent warm
// runs can replay them without re-fetching comments. Tasks not in mentionsByTask
// are stored with an empty mentions slice (we know they have none).
func updateCacheFromTasks(cache *inboxCache, tasks []clickup.Task, mentionsByTask map[string][]mention, now time.Time) {
	if cache.Tasks == nil {
		cache.Tasks = map[string]inboxCacheEntry{}
	}
	for _, t := range tasks {
		cache.Tasks[t.ID] = inboxCacheEntry{
			DateUpdated: t.DateUpdated,
			Mentions:    mentionsByTask[t.ID],
		}
	}
	cache.ScannedAt = now.UnixMilli()
}

// cachedMentionsFor returns the mentions stored in the cache for tasks the
// caller chose to skip (i.e. tasks whose date_updated matched the cache).
func cachedMentionsFor(cache *inboxCache, skipped []clickup.Task) []mention {
	if cache == nil {
		return nil
	}
	var out []mention
	for _, t := range skipped {
		if entry, ok := cache.Tasks[t.ID]; ok {
			out = append(out, entry.Mentions...)
		}
	}
	return out
}

// skippedTasks returns the inverse of cacheDiff: tasks present in the workspace
// scan whose comments we did not refetch this run.
func skippedTasks(cache *inboxCache, all []clickup.Task) []clickup.Task {
	if cache == nil || len(cache.Tasks) == 0 {
		return nil
	}
	var skipped []clickup.Task
	for _, t := range all {
		if entry, ok := cache.Tasks[t.ID]; ok && entry.DateUpdated == t.DateUpdated {
			skipped = append(skipped, t)
		}
	}
	return skipped
}

// isNewAssignment reports whether a task represents a fresh assignment to userID:
// the user is in the assignee list and the task was created within the lookback
// window. ClickUp does not expose an API to detect re-assignments of pre-existing
// tasks, so this is our best heuristic for "newly assigned to me".
func isNewAssignment(task clickup.Task, userID int, cutoffMs int64) bool {
	createdMs, err := strconv.ParseInt(task.DateCreated, 10, 64)
	if err != nil {
		return false
	}
	if createdMs < cutoffMs {
		return false
	}
	for _, a := range task.Assignees {
		if a.ID == userID {
			return true
		}
	}
	return false
}

// eventKey returns a composite key used to dedupe events. Comment mentions are
// keyed by comment ID so multiple distinct mentions on the same task survive,
// while assignment and description-mention events have one slot per task.
func eventKey(e mention) string {
	return e.Type + "|" + e.TaskID + "|" + e.CommentID
}

// mergeEvents combines one or more event slices, dedupes by composite key, and
// sorts the result oldest-first so the newest entries land at the bottom of the
// terminal output (closest to the cursor).
func mergeEvents(groups ...[]mention) []mention {
	seen := map[string]bool{}
	var out []mention
	for _, group := range groups {
		for _, e := range group {
			key := eventKey(e)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DateMs < out[j].DateMs
	})
	return out
}
