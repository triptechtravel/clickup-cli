package inbox

import (
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/raksul/go-clickup/clickup"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

func TestNewCmdInbox_Defaults(t *testing.T) {
	f := &cmdutil.Factory{}
	cmd := NewCmdInbox(f)

	daysFlag := cmd.Flags().Lookup("days")
	if daysFlag == nil {
		t.Fatal("expected --days flag")
	}
	if daysFlag.DefValue != "7" {
		t.Errorf("--days default = %q, want %q", daysFlag.DefValue, "7")
	}

	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("expected --limit flag")
	}
	if limitFlag.DefValue != "200" {
		t.Errorf("--limit default = %q, want %q", limitFlag.DefValue, "200")
	}
}

func TestContainsMention_TaskDescriptions(t *testing.T) {
	// The same containsMention function is used for both comments and task
	// descriptions. These cases test description-specific patterns.
	tests := []struct {
		name     string
		desc     string
		username string
		want     bool
	}{
		{
			name:     "mention in multiline description",
			desc:     "This task requires\n@alice to review the approach\nbefore we proceed.",
			username: "alice",
			want:     true,
		},
		{
			name:     "mention with full name style",
			desc:     "Assigned to @isaac rowntree for implementation",
			username: "isaac rowntree",
			want:     true,
		},
		{
			name:     "no mention in description",
			desc:     "Implement the new feature for the dashboard",
			username: "alice",
			want:     false,
		},
		{
			name:     "empty description",
			desc:     "",
			username: "alice",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsMention(tt.desc, tt.username)
			if got != tt.want {
				t.Errorf("containsMention(%q, %q) = %v, want %v",
					tt.desc, tt.username, got, tt.want)
			}
		})
	}
}

func TestContainsMention(t *testing.T) {
	tests := []struct {
		name        string
		commentText string
		username    string
		want        bool
	}{
		{
			name:        "exact @username mention",
			commentText: "Hey @alice can you review this?",
			username:    "alice",
			want:        true,
		},
		{
			name:        "@ username with space",
			commentText: "Hey @ alice can you review this?",
			username:    "alice",
			want:        true,
		},
		{
			name:        "case insensitive - uppercase in comment",
			commentText: "Hey @ALICE can you review this?",
			username:    "alice",
			want:        true,
		},
		{
			name:        "username must be lowercase - uppercase username does not match",
			commentText: "Hey @alice can you review this?",
			username:    "ALICE",
			want:        false,
		},
		{
			name:        "case insensitive - mixed case",
			commentText: "Hey @Alice can you review this?",
			username:    "alice",
			want:        true,
		},
		{
			name:        "no match - different username",
			commentText: "Hey @bob can you review this?",
			username:    "alice",
			want:        false,
		},
		{
			name:        "no match - no @ sign",
			commentText: "Hey alice can you review this?",
			username:    "alice",
			want:        false,
		},
		{
			name:        "no match - empty comment",
			commentText: "",
			username:    "alice",
			want:        false,
		},
		{
			name:        "mention at beginning of string",
			commentText: "@alice please check",
			username:    "alice",
			want:        true,
		},
		{
			name:        "mention at end of string",
			commentText: "Please check @alice",
			username:    "alice",
			want:        true,
		},
		{
			name:        "mention with space at beginning",
			commentText: "@ alice please check",
			username:    "alice",
			want:        true,
		},
		{
			name:        "username is substring of another - still matches",
			commentText: "Hey @alicewonderland what's up?",
			username:    "alice",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsMention(tt.commentText, tt.username)
			if got != tt.want {
				t.Errorf("containsMention(%q, %q) = %v, want %v",
					tt.commentText, tt.username, got, tt.want)
			}
		})
	}
}

func TestFormatMentionDate(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		// We check the output is non-empty and not the raw input for valid timestamps,
		// or equals the raw input for invalid ones.
		wantRaw bool // if true, expect the raw dateStr back
	}{
		{
			name:    "valid timestamp - returns relative time string",
			dateStr: strconv.FormatInt(time.Now().Add(-2*time.Hour).UnixMilli(), 10),
			wantRaw: false,
		},
		{
			name:    "valid timestamp - recent",
			dateStr: strconv.FormatInt(time.Now().Add(-30*time.Second).UnixMilli(), 10),
			wantRaw: false,
		},
		{
			name:    "invalid - not a number",
			dateStr: "not-a-number",
			wantRaw: true,
		},
		{
			name:    "invalid - empty string",
			dateStr: "",
			wantRaw: true,
		},
		{
			name:    "invalid - letters mixed with numbers",
			dateStr: "123abc456",
			wantRaw: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMentionDate(tt.dateStr)
			if tt.wantRaw {
				if got != tt.dateStr {
					t.Errorf("formatMentionDate(%q) = %q, want raw input %q", tt.dateStr, got, tt.dateStr)
				}
			} else {
				// For valid timestamps, the result should be a relative time string,
				// not the original numeric string.
				if got == tt.dateStr {
					t.Errorf("formatMentionDate(%q) = %q, expected a relative time string", tt.dateStr, got)
				}
				if got == "" {
					t.Errorf("formatMentionDate(%q) returned empty string", tt.dateStr)
				}
			}
		})
	}
}

func TestExtractAttachmentURLs(t *testing.T) {
	tests := []struct {
		name   string
		blocks []commentBlock
		want   int
	}{
		{
			name:   "no blocks",
			blocks: nil,
			want:   0,
		},
		{
			name: "text only blocks",
			blocks: []commentBlock{
				{Text: "hello"},
				{Text: "world"},
			},
			want: 0,
		},
		{
			name: "image attachment",
			blocks: []commentBlock{
				{Type: "image", Image: &commentMediaObject{URL: "https://example.com/image.png"}},
			},
			want: 1,
		},
		{
			name: "frame attachment",
			blocks: []commentBlock{
				{Type: "frame", Frame: &commentMediaObject{URL: "https://example.com/video.mp4"}},
			},
			want: 1,
		},
		{
			name: "mixed blocks",
			blocks: []commentBlock{
				{Text: "Check this out"},
				{Type: "image", Image: &commentMediaObject{URL: "https://example.com/a.png"}},
				{Text: "and also this"},
				{Type: "frame", Frame: &commentMediaObject{URL: "https://example.com/b.mp4"}},
				{Type: "image", Image: &commentMediaObject{URL: "https://example.com/c.png"}},
			},
			want: 3,
		},
		{
			name: "image type but nil image object",
			blocks: []commentBlock{
				{Type: "image", Image: nil},
			},
			want: 0,
		},
		{
			name: "image type but empty URL",
			blocks: []commentBlock{
				{Type: "image", Image: &commentMediaObject{URL: ""}},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAttachmentURLs(tt.blocks)
			if len(got) != tt.want {
				t.Errorf("extractAttachmentURLs() returned %d URLs, want %d", len(got), tt.want)
			}
		})
	}
}

func TestFormatMentionDate_KnownValue(t *testing.T) {
	// Use a timestamp exactly 2 hours ago. RelativeTime should return "2 hours ago".
	ts := time.Now().Add(-2 * time.Hour).UnixMilli()
	dateStr := strconv.FormatInt(ts, 10)

	got := formatMentionDate(dateStr)
	if got != "2 hours ago" {
		t.Errorf("formatMentionDate(%q) = %q, want %q", dateStr, got, "2 hours ago")
	}
}

// ----- inbox cache tests -----

func TestLoadInboxCache_Missing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	c, err := loadInboxCache(path)
	if err != nil {
		t.Fatalf("loadInboxCache on missing file returned error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
	if c.Tasks == nil {
		t.Error("expected non-nil Tasks map")
	}
	if len(c.Tasks) != 0 {
		t.Errorf("expected empty Tasks map, got %d entries", len(c.Tasks))
	}
}

func TestLoadInboxCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	original := &inboxCache{
		ScannedAt: 1700000000000,
		Tasks: map[string]inboxCacheEntry{
			"abc123": {DateUpdated: "1699999999000"},
			"def456": {DateUpdated: "1699999998000"},
		},
	}
	if err := saveInboxCache(path, original); err != nil {
		t.Fatalf("saveInboxCache: %v", err)
	}

	loaded, err := loadInboxCache(path)
	if err != nil {
		t.Fatalf("loadInboxCache: %v", err)
	}

	if loaded.ScannedAt != original.ScannedAt {
		t.Errorf("ScannedAt: got %d, want %d", loaded.ScannedAt, original.ScannedAt)
	}
	if len(loaded.Tasks) != len(original.Tasks) {
		t.Errorf("Tasks length: got %d, want %d", len(loaded.Tasks), len(original.Tasks))
	}
	for id, entry := range original.Tasks {
		got, ok := loaded.Tasks[id]
		if !ok {
			t.Errorf("missing task %q in loaded cache", id)
			continue
		}
		if got.DateUpdated != entry.DateUpdated {
			t.Errorf("task %q DateUpdated: got %q, want %q", id, got.DateUpdated, entry.DateUpdated)
		}
	}
}

func TestSaveInboxCache_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "subdir", "cache.json")

	c := &inboxCache{ScannedAt: 1, Tasks: map[string]inboxCacheEntry{}}
	if err := saveInboxCache(path, c); err != nil {
		t.Fatalf("saveInboxCache should create parent dirs: %v", err)
	}
}

func TestInboxCache_IsFresh(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		scannedAt int64
		want      bool
	}{
		{"never scanned", 0, false},
		{"just scanned", now.UnixMilli(), true},
		{"scanned 1h ago", now.Add(-1 * time.Hour).UnixMilli(), true},
		{"scanned 23h ago", now.Add(-23 * time.Hour).UnixMilli(), true},
		{"scanned 25h ago - stale", now.Add(-25 * time.Hour).UnixMilli(), false},
		{"scanned 1 week ago - stale", now.Add(-7 * 24 * time.Hour).UnixMilli(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &inboxCache{ScannedAt: tt.scannedAt}
			got := c.IsFresh(now)
			if got != tt.want {
				t.Errorf("IsFresh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheDiff_EmptyCacheReturnsAll(t *testing.T) {
	cache := &inboxCache{Tasks: map[string]inboxCacheEntry{}}
	tasks := []clickup.Task{
		{ID: "a", DateUpdated: "100"},
		{ID: "b", DateUpdated: "200"},
	}

	got := cacheDiff(cache, tasks)
	if len(got) != 2 {
		t.Errorf("expected all 2 tasks returned, got %d", len(got))
	}
}

func TestCacheDiff_NilCacheReturnsAll(t *testing.T) {
	tasks := []clickup.Task{
		{ID: "a", DateUpdated: "100"},
	}

	got := cacheDiff(nil, tasks)
	if len(got) != 1 {
		t.Errorf("expected all tasks returned for nil cache, got %d", len(got))
	}
}

func TestCacheDiff_UnchangedTasksSkipped(t *testing.T) {
	cache := &inboxCache{
		Tasks: map[string]inboxCacheEntry{
			"a": {DateUpdated: "100"},
			"b": {DateUpdated: "200"},
		},
	}
	tasks := []clickup.Task{
		{ID: "a", DateUpdated: "100"},
		{ID: "b", DateUpdated: "200"},
	}

	got := cacheDiff(cache, tasks)
	if len(got) != 0 {
		t.Errorf("expected 0 tasks (all unchanged), got %d", len(got))
	}
}

func TestCacheDiff_ChangedTaskIncluded(t *testing.T) {
	cache := &inboxCache{
		Tasks: map[string]inboxCacheEntry{
			"a": {DateUpdated: "100"},
			"b": {DateUpdated: "200"},
		},
	}
	tasks := []clickup.Task{
		{ID: "a", DateUpdated: "100"}, // unchanged
		{ID: "b", DateUpdated: "250"}, // updated
	}

	got := cacheDiff(cache, tasks)
	if len(got) != 1 {
		t.Fatalf("expected 1 task in diff, got %d", len(got))
	}
	if got[0].ID != "b" {
		t.Errorf("expected task b in diff, got %q", got[0].ID)
	}
}

func TestCacheDiff_NewTaskIncluded(t *testing.T) {
	cache := &inboxCache{
		Tasks: map[string]inboxCacheEntry{
			"a": {DateUpdated: "100"},
		},
	}
	tasks := []clickup.Task{
		{ID: "a", DateUpdated: "100"}, // known
		{ID: "c", DateUpdated: "300"}, // never seen
	}

	got := cacheDiff(cache, tasks)
	if len(got) != 1 {
		t.Fatalf("expected 1 task in diff, got %d", len(got))
	}
	if got[0].ID != "c" {
		t.Errorf("expected task c in diff, got %q", got[0].ID)
	}
}

func TestUpdateCacheFromTasks(t *testing.T) {
	cache := &inboxCache{Tasks: map[string]inboxCacheEntry{
		"old": {DateUpdated: "50"}, // pre-existing entry should be preserved
	}}
	tasks := []clickup.Task{
		{ID: "a", DateUpdated: "100"},
		{ID: "b", DateUpdated: "200"},
	}
	mentionsByTask := map[string][]mention{
		"a": {{TaskID: "a", Type: eventCommentMention, CommentID: "c1", DateMs: 90}},
	}
	now := time.UnixMilli(1700000000000)

	updateCacheFromTasks(cache, tasks, mentionsByTask, now)

	if cache.ScannedAt != now.UnixMilli() {
		t.Errorf("ScannedAt: got %d, want %d", cache.ScannedAt, now.UnixMilli())
	}
	if got := cache.Tasks["a"].DateUpdated; got != "100" {
		t.Errorf("task a DateUpdated: got %q, want %q", got, "100")
	}
	if got := len(cache.Tasks["a"].Mentions); got != 1 {
		t.Errorf("task a Mentions length: got %d, want 1", got)
	}
	if got := cache.Tasks["b"].DateUpdated; got != "200" {
		t.Errorf("task b DateUpdated: got %q, want %q", got, "200")
	}
	if got := len(cache.Tasks["b"].Mentions); got != 0 {
		t.Errorf("task b should have 0 mentions, got %d", got)
	}
	if got := cache.Tasks["old"].DateUpdated; got != "50" {
		t.Errorf("pre-existing entry should be preserved, got %q", got)
	}
}

func TestUpdateCacheFromTasks_NilTasksMap(t *testing.T) {
	cache := &inboxCache{} // Tasks map is nil
	tasks := []clickup.Task{{ID: "a", DateUpdated: "100"}}

	updateCacheFromTasks(cache, tasks, nil, time.Now())

	if cache.Tasks == nil {
		t.Fatal("Tasks map should be initialized")
	}
	if cache.Tasks["a"].DateUpdated != "100" {
		t.Errorf("expected entry for task a")
	}
}

func TestSkippedTasks_ReturnsUnchanged(t *testing.T) {
	cache := &inboxCache{Tasks: map[string]inboxCacheEntry{
		"a": {DateUpdated: "100"},
		"b": {DateUpdated: "200"},
	}}
	all := []clickup.Task{
		{ID: "a", DateUpdated: "100"}, // unchanged → skipped
		{ID: "b", DateUpdated: "250"}, // changed → not skipped
		{ID: "c", DateUpdated: "300"}, // new → not skipped
	}

	got := skippedTasks(cache, all)
	if len(got) != 1 || got[0].ID != "a" {
		t.Errorf("expected only task a to be skipped, got %+v", got)
	}
}

func TestSkippedTasks_NilCache(t *testing.T) {
	all := []clickup.Task{{ID: "a", DateUpdated: "100"}}
	got := skippedTasks(nil, all)
	if len(got) != 0 {
		t.Errorf("nil cache should produce 0 skipped, got %d", len(got))
	}
}

func TestCachedMentionsFor_ReplaysStoredMentions(t *testing.T) {
	cache := &inboxCache{Tasks: map[string]inboxCacheEntry{
		"a": {
			DateUpdated: "100",
			Mentions: []mention{
				{TaskID: "a", Type: eventCommentMention, CommentID: "c1", DateMs: 50},
				{TaskID: "a", Type: eventDescriptionMention, DateMs: 40},
			},
		},
		"b": {DateUpdated: "200"}, // no stored mentions
	}}
	skipped := []clickup.Task{
		{ID: "a", DateUpdated: "100"},
		{ID: "b", DateUpdated: "200"},
	}

	got := cachedMentionsFor(cache, skipped)
	if len(got) != 2 {
		t.Errorf("expected 2 cached mentions, got %d", len(got))
	}
}

func TestCachedMentionsFor_NilCache(t *testing.T) {
	skipped := []clickup.Task{{ID: "a", DateUpdated: "100"}}
	got := cachedMentionsFor(nil, skipped)
	if len(got) != 0 {
		t.Errorf("nil cache should produce 0 mentions, got %d", len(got))
	}
}

// Round-trip integration: cold scan stores mentions; warm scan replays them.
func TestInboxCache_WarmRunReplaysMentions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	tasks := []clickup.Task{
		{ID: "a", DateUpdated: "100"},
		{ID: "b", DateUpdated: "200"},
	}
	mentionsByTask := map[string][]mention{
		"a": {{TaskID: "a", Type: eventCommentMention, CommentID: "c1", DateMs: 90}},
		"b": {{TaskID: "b", Type: eventCommentMention, CommentID: "c2", DateMs: 180}},
	}

	// Cold run: build cache and save.
	cold := &inboxCache{Tasks: map[string]inboxCacheEntry{}}
	updateCacheFromTasks(cold, tasks, mentionsByTask, time.Now())
	if err := saveInboxCache(path, cold); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Warm run: load, diff, replay.
	warm, err := loadInboxCache(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	diff := cacheDiff(warm, tasks)
	if len(diff) != 0 {
		t.Errorf("expected 0 tasks in diff, got %d", len(diff))
	}
	skipped := skippedTasks(warm, tasks)
	replayed := cachedMentionsFor(warm, skipped)
	if len(replayed) != 2 {
		t.Errorf("expected 2 replayed mentions, got %d", len(replayed))
	}
}

// ----- assignment classification tests -----

func TestIsNewAssignment_NewlyCreatedAndAssigned(t *testing.T) {
	cutoff := time.Now().Add(-7 * 24 * time.Hour).UnixMilli()
	createdMs := time.Now().Add(-2 * 24 * time.Hour).UnixMilli()

	task := clickup.Task{
		DateCreated: strconv.FormatInt(createdMs, 10),
		Assignees: []clickup.User{
			{ID: 42},
			{ID: 99},
		},
	}

	if !isNewAssignment(task, 42, cutoff) {
		t.Error("expected task to be classified as new assignment")
	}
}

func TestIsNewAssignment_OldTask(t *testing.T) {
	cutoff := time.Now().Add(-7 * 24 * time.Hour).UnixMilli()
	createdMs := time.Now().Add(-30 * 24 * time.Hour).UnixMilli() // 30 days ago

	task := clickup.Task{
		DateCreated: strconv.FormatInt(createdMs, 10),
		Assignees:   []clickup.User{{ID: 42}},
	}

	if isNewAssignment(task, 42, cutoff) {
		t.Error("old task should not be classified as new assignment")
	}
}

func TestIsNewAssignment_UserNotAssigned(t *testing.T) {
	cutoff := time.Now().Add(-7 * 24 * time.Hour).UnixMilli()
	createdMs := time.Now().Add(-1 * 24 * time.Hour).UnixMilli()

	task := clickup.Task{
		DateCreated: strconv.FormatInt(createdMs, 10),
		Assignees:   []clickup.User{{ID: 99}}, // not user 42
	}

	if isNewAssignment(task, 42, cutoff) {
		t.Error("task without user as assignee should not be classified")
	}
}

func TestIsNewAssignment_NoAssignees(t *testing.T) {
	cutoff := time.Now().Add(-7 * 24 * time.Hour).UnixMilli()
	createdMs := time.Now().Add(-1 * 24 * time.Hour).UnixMilli()

	task := clickup.Task{
		DateCreated: strconv.FormatInt(createdMs, 10),
		Assignees:   nil,
	}

	if isNewAssignment(task, 42, cutoff) {
		t.Error("task with no assignees should not be classified")
	}
}

func TestIsNewAssignment_InvalidDate(t *testing.T) {
	cutoff := time.Now().Add(-7 * 24 * time.Hour).UnixMilli()

	task := clickup.Task{
		DateCreated: "not-a-number",
		Assignees:   []clickup.User{{ID: 42}},
	}

	if isNewAssignment(task, 42, cutoff) {
		t.Error("task with invalid date should not be classified")
	}
}

// ----- merge / dedup tests -----

func TestMergeEvents_DedupesByKey(t *testing.T) {
	events := [][]mention{
		{
			{TaskID: "a", Type: eventCommentMention, CommentID: "c1", DateMs: 100},
			{TaskID: "a", Type: eventCommentMention, CommentID: "c1", DateMs: 100},
		},
	}

	got := mergeEvents(events...)
	if len(got) != 1 {
		t.Errorf("expected 1 event after dedup, got %d", len(got))
	}
}

func TestMergeEvents_DistinctTypesNotDedupedSameTask(t *testing.T) {
	events := [][]mention{
		{
			{TaskID: "a", Type: eventAssignment, DateMs: 100},
			{TaskID: "a", Type: eventCommentMention, CommentID: "c1", DateMs: 200},
			{TaskID: "a", Type: eventDescriptionMention, DateMs: 150},
		},
	}

	got := mergeEvents(events...)
	if len(got) != 3 {
		t.Errorf("expected 3 distinct events for same task, got %d", len(got))
	}
}

func TestMergeEvents_DedupAcrossGroups(t *testing.T) {
	groupA := []mention{
		{TaskID: "a", Type: eventAssignment, DateMs: 100},
	}
	groupB := []mention{
		{TaskID: "a", Type: eventAssignment, DateMs: 100}, // duplicate
		{TaskID: "b", Type: eventCommentMention, CommentID: "c1", DateMs: 200},
	}

	got := mergeEvents(groupA, groupB)
	if len(got) != 2 {
		t.Errorf("expected 2 unique events, got %d", len(got))
	}
}

func TestMergeEvents_SortedOldestFirst(t *testing.T) {
	events := [][]mention{
		{
			{TaskID: "a", Type: eventAssignment, DateMs: 300},
			{TaskID: "b", Type: eventCommentMention, CommentID: "c1", DateMs: 100},
			{TaskID: "c", Type: eventDescriptionMention, DateMs: 200},
		},
	}

	got := mergeEvents(events...)
	if len(got) != 3 {
		t.Fatalf("expected 3 events, got %d", len(got))
	}

	if !sort.SliceIsSorted(got, func(i, j int) bool {
		return got[i].DateMs < got[j].DateMs
	}) {
		t.Errorf("expected events sorted oldest-first, got %+v", got)
	}
	if got[0].DateMs != 100 || got[2].DateMs != 300 {
		t.Errorf("sort order wrong: got %d, %d, %d", got[0].DateMs, got[1].DateMs, got[2].DateMs)
	}
}

func TestMergeEvents_EmptyInputs(t *testing.T) {
	got := mergeEvents()
	if len(got) != 0 {
		t.Errorf("expected 0 events for empty input, got %d", len(got))
	}

	got = mergeEvents(nil, nil)
	if len(got) != 0 {
		t.Errorf("expected 0 events for nil groups, got %d", len(got))
	}
}

func TestEventKey_DistinctForDifferentTypes(t *testing.T) {
	a := mention{TaskID: "x", Type: eventAssignment}
	b := mention{TaskID: "x", Type: eventDescriptionMention}
	c := mention{TaskID: "x", Type: eventCommentMention, CommentID: "c1"}

	if eventKey(a) == eventKey(b) {
		t.Error("assignment and description_mention should have different keys")
	}
	if eventKey(b) == eventKey(c) {
		t.Error("description_mention and comment_mention should have different keys")
	}
	if eventKey(a) == eventKey(c) {
		t.Error("assignment and comment_mention should have different keys")
	}
}

func TestEventKey_SameForDuplicateCommentMentions(t *testing.T) {
	a := mention{TaskID: "x", Type: eventCommentMention, CommentID: "c1"}
	b := mention{TaskID: "x", Type: eventCommentMention, CommentID: "c1"}

	if eventKey(a) != eventKey(b) {
		t.Errorf("identical comment mentions should have same key: %q vs %q", eventKey(a), eventKey(b))
	}
}

func TestNewCmdInbox_NoCacheFlag(t *testing.T) {
	f := &cmdutil.Factory{}
	cmd := NewCmdInbox(f)

	flag := cmd.Flags().Lookup("no-cache")
	if flag == nil {
		t.Fatal("expected --no-cache flag")
	}
	if flag.DefValue != "false" {
		t.Errorf("--no-cache default = %q, want %q", flag.DefValue, "false")
	}
}
