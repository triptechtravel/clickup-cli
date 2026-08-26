package task

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

func TestNewCmdSearch_Flags(t *testing.T) {
	cmd := NewCmdSearch(nil)

	assert.NotNil(t, cmd.Flags().Lookup("space"))
	assert.NotNil(t, cmd.Flags().Lookup("folder"))
	assert.NotNil(t, cmd.Flags().Lookup("assignee"))
	assert.NotNil(t, cmd.Flags().Lookup("pick"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.Equal(t, "search [query]", cmd.Use)
}

func TestNewCmdSearch_CommentsFlag(t *testing.T) {
	cmd := NewCmdSearch(nil)
	assert.NotNil(t, cmd.Flags().Lookup("comments"))
}

func TestScoreTaskName_Substring(t *testing.T) {
	kind, rank, ok := scoreTaskName("geozone", "Geozone schema updates")
	assert.True(t, ok)
	assert.Equal(t, matchSubstring, kind)
	assert.Equal(t, 0, rank)
}

func TestScoreTaskName_NoMatch(t *testing.T) {
	_, _, ok := scoreTaskName("xyz123abc", "Totally different task")
	assert.False(t, ok)
}

func TestScoreTaskName_DescriptionMatch(t *testing.T) {
	tasks := []searchTask{
		{ID: "1", Name: "Unrelated task name", Description: "This task involves a geozone migration"},
	}

	matched, unmatched := filterTasks("geozone", tasks)
	assert.Len(t, matched, 1)
	assert.Equal(t, matchDescription, matched[0].kind)
	assert.Empty(t, unmatched)
}

func TestFilterTasks_NameBeatsDescription(t *testing.T) {
	tasks := []searchTask{
		{ID: "1", Name: "Geozone schema updates", Description: "This also mentions geozone"},
	}

	matched, _ := filterTasks("geozone", tasks)
	assert.Len(t, matched, 1)
	assert.Equal(t, matchSubstring, matched[0].kind) // name match takes priority
}

func TestFilterTasks_DescriptionFallback(t *testing.T) {
	tasks := []searchTask{
		{ID: "1", Name: "Update database schema", Description: "Migrate geozone tables to new format"},
		{ID: "2", Name: "Fix login bug", Description: "Users cannot log in properly"},
		{ID: "3", Name: "Geozone v2", Description: "New geozone implementation"},
	}

	matched, unmatched := filterTasks("geozone", tasks)

	// Task 1: description match, Task 3: name match, Task 2: no match
	assert.Len(t, matched, 2)
	assert.Len(t, unmatched, 1)
	assert.Equal(t, "2", unmatched[0].ID)

	// After sorting, name match should come first
	sortScoredTasks(matched)
	assert.Equal(t, "3", matched[0].ID) // name substring
	assert.Equal(t, matchSubstring, matched[0].kind)
	assert.Equal(t, "1", matched[1].ID) // description
	assert.Equal(t, matchDescription, matched[1].kind)
}

// ---------------------------------------------------------------------------
// sortScoredTasks
// ---------------------------------------------------------------------------

func TestSortScoredTasks_MixedKinds(t *testing.T) {
	tasks := []scoredTask{
		{searchTask: searchTask{ID: "1"}, kind: matchComment, fuzzyRank: 0},
		{searchTask: searchTask{ID: "2"}, kind: matchSubstring, fuzzyRank: 0},
		{searchTask: searchTask{ID: "3"}, kind: matchDescription, fuzzyRank: 0},
		{searchTask: searchTask{ID: "4"}, kind: matchFuzzy, fuzzyRank: 5},
	}

	sortScoredTasks(tasks)

	assert.Equal(t, matchSubstring, tasks[0].kind)
	assert.Equal(t, "2", tasks[0].ID)
	assert.Equal(t, matchFuzzy, tasks[1].kind)
	assert.Equal(t, "4", tasks[1].ID)
	assert.Equal(t, matchDescription, tasks[2].kind)
	assert.Equal(t, "3", tasks[2].ID)
	assert.Equal(t, matchComment, tasks[3].kind)
	assert.Equal(t, "1", tasks[3].ID)
}

func TestSortScoredTasks_SameKindFuzzyOrdering(t *testing.T) {
	tasks := []scoredTask{
		{searchTask: searchTask{ID: "1"}, kind: matchFuzzy, fuzzyRank: 10},
		{searchTask: searchTask{ID: "2"}, kind: matchFuzzy, fuzzyRank: 2},
		{searchTask: searchTask{ID: "3"}, kind: matchFuzzy, fuzzyRank: 5},
	}

	sortScoredTasks(tasks)

	assert.Equal(t, "2", tasks[0].ID) // rank 2 (best)
	assert.Equal(t, "3", tasks[1].ID) // rank 5
	assert.Equal(t, "1", tasks[2].ID) // rank 10 (worst)
}

// ---------------------------------------------------------------------------
// dedupScored
// ---------------------------------------------------------------------------

func TestDedupScored_KeepBestKind(t *testing.T) {
	tasks := []scoredTask{
		{searchTask: searchTask{ID: "1"}, kind: matchDescription, fuzzyRank: 0},
		{searchTask: searchTask{ID: "1"}, kind: matchSubstring, fuzzyRank: 0},
		{searchTask: searchTask{ID: "2"}, kind: matchComment, fuzzyRank: 0},
	}

	result := dedupScored(tasks)

	assert.Len(t, result, 2)
	// ID "1" should keep matchSubstring (lower kind = better)
	for _, r := range result {
		if r.ID == "1" {
			assert.Equal(t, matchSubstring, r.kind)
		}
	}
}

func TestDedupScored_KeepBestFuzzyRankForSameKind(t *testing.T) {
	tasks := []scoredTask{
		{searchTask: searchTask{ID: "1"}, kind: matchFuzzy, fuzzyRank: 10},
		{searchTask: searchTask{ID: "1"}, kind: matchFuzzy, fuzzyRank: 3},
	}

	result := dedupScored(tasks)

	assert.Len(t, result, 1)
	assert.Equal(t, 3, result[0].fuzzyRank) // keeps better rank
}

func TestDedupScored_PreservesOrder(t *testing.T) {
	tasks := []scoredTask{
		{searchTask: searchTask{ID: "3"}, kind: matchSubstring},
		{searchTask: searchTask{ID: "1"}, kind: matchSubstring},
		{searchTask: searchTask{ID: "2"}, kind: matchSubstring},
	}

	result := dedupScored(tasks)

	assert.Len(t, result, 3)
	assert.Equal(t, "3", result[0].ID)
	assert.Equal(t, "1", result[1].ID)
	assert.Equal(t, "2", result[2].ID)
}

// ---------------------------------------------------------------------------
// resolveAssignee helpers
// ---------------------------------------------------------------------------

// teamsJSON returns a JSON body for GET /team with the given members.
func teamsJSON(members ...struct {
	ID       int
	Username string
}) string {
	type user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}
	type memberObj struct {
		User user `json:"user"`
	}
	type team struct {
		ID      string      `json:"id"`
		Name    string      `json:"name"`
		Members []memberObj `json:"members"`
	}
	var ms []memberObj
	for _, m := range members {
		ms = append(ms, memberObj{User: user{ID: m.ID, Username: m.Username}})
	}
	b, _ := json.Marshal(struct {
		Teams []team `json:"teams"`
	}{
		Teams: []team{{ID: "12345", Name: "Test Workspace", Members: ms}},
	})
	return string(b)
}

func makeMember(id int, username string) struct {
	ID       int
	Username string
} {
	return struct {
		ID       int
		Username string
	}{ID: id, Username: username}
}

func setupTeamAndUser(tf *testutil.TestFactory, currentUserID int, members ...struct {
	ID       int
	Username string
}) {
	tf.Handle("GET", "team", 200, teamsJSON(members...))
	tf.Handle("GET", "user", 200, fmt.Sprintf(`{"user":{"id":%d}}`, currentUserID))
}

// ---------------------------------------------------------------------------
// resolveAssignee tests
// ---------------------------------------------------------------------------

func TestResolveAssignee_Me(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	setupTeamAndUser(tf, 100,
		makeMember(100, "isaac"),
		makeMember(200, "alice"),
	)

	client, _ := tf.Factory.ApiClient()
	id, name, err := resolveAssignee(t.Context(), client, "me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.Equal(t, 100, id)
	assert.Equal(t, "isaac", name)
}

func TestResolveAssignee_NumericID(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	setupTeamAndUser(tf, 100,
		makeMember(100, "isaac"),
		makeMember(54695018, "bob"),
	)

	client, _ := tf.Factory.ApiClient()
	id, name, err := resolveAssignee(t.Context(), client, "54695018")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.Equal(t, 54695018, id)
	assert.Equal(t, "bob", name)
}

func TestResolveAssignee_ExactUsername(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	setupTeamAndUser(tf, 100,
		makeMember(100, "Isaac"),
		makeMember(200, "Alice"),
	)

	client, _ := tf.Factory.ApiClient()
	id, name, err := resolveAssignee(t.Context(), client, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.Equal(t, 200, id)
	assert.Equal(t, "Alice", name)
}

func TestResolveAssignee_Substring(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	setupTeamAndUser(tf, 100,
		makeMember(100, "Isaac Rowntree"),
		makeMember(200, "Alice Wonder"),
	)

	client, _ := tf.Factory.ApiClient()
	id, name, err := resolveAssignee(t.Context(), client, "Rown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.Equal(t, 100, id)
	assert.Equal(t, "Isaac Rowntree", name)
}

func TestResolveAssignee_Ambiguous(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	setupTeamAndUser(tf, 100,
		makeMember(100, "Isaac Smith"),
		makeMember(200, "Isaac Jones"),
	)

	client, _ := tf.Factory.ApiClient()
	_, _, err := resolveAssignee(t.Context(), client, "Isaac")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous match")
	assert.Contains(t, err.Error(), "Isaac Smith")
	assert.Contains(t, err.Error(), "Isaac Jones")
}

func TestResolveAssignee_NotFound(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	setupTeamAndUser(tf, 100,
		makeMember(100, "Isaac"),
		makeMember(200, "Alice"),
	)

	client, _ := tf.Factory.ApiClient()
	_, _, err := resolveAssignee(t.Context(), client, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workspace member found")
}

// ---------------------------------------------------------------------------
// Workspace fetch
// ---------------------------------------------------------------------------

// This test used to assert that `search=Bug` reached the API. It does not any
// more: ClickUp ignores that param, and sending it made a client-side filter
// look like a server-side one. TestSearch_OmitsIgnoredServerSideSearchParam
// now pins the opposite. What is left here is what always mattered — a match
// in the fetched page reaches the output.
func TestSearch_MatchFromWorkspaceReachesOutput(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleFunc("team/12345/task", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"tasks":[{"id":"abc","name":"Server Bug Fix","status":{"status":"open"},"assignees":[]}]}`))
	})
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "Bug", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := tf.OutBuf.String()
	assert.Contains(t, out, "abc")
	assert.Contains(t, out, "Server Bug Fix")
}

func TestSearchIncludeSubtasks(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedURL string
	tf.HandleFunc("team/12345/task", func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"tasks":[{"id":"abc","name":"Server Bug Fix","status":{"status":"open"},"assignees":[]}]}`))
	})

	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	cmd := NewCmdSearch(tf.Factory)
	err := testutil.RunCommand(t, cmd, "Bug", "--include-subtasks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.True(t, strings.Contains(capturedURL, "subtasks=true"),
		"expected subtasks=true in URL, got: %s", capturedURL)
}

// ---------------------------------------------------------------------------
// Workspace sweep
//
// ClickUp has no server-side text search: the `search=` param on
// GET team/{id}/task is accepted and then ignored, returning the same
// date-ordered page regardless of the query. Every match is therefore found by
// filtering full pages client-side, and any design that stops at the first
// tier returning *something* will silently drop better matches living further
// in. These tests pin the sweep to "read every page, return every match".
// ---------------------------------------------------------------------------

// searchTasksJSON builds a team/{id}/task response body from id/name pairs.
func searchTasksJSON(pairs ...[2]string) string {
	list := make([]map[string]any, 0, len(pairs))
	for _, p := range pairs {
		list = append(list, map[string]any{
			"id":        p[0],
			"name":      p[1],
			"status":    map[string]any{"status": "open"},
			"assignees": []any{},
		})
	}
	b, _ := json.Marshal(map[string]any{"tasks": list})
	return string(b)
}

// fillerPairs returns n non-matching id/name pairs, enough to make a page look
// full so the sweep keeps paginating.
func fillerPairs(n int) [][2]string {
	out := make([][2]string, n)
	for i := range out {
		out[i] = [2]string{fmt.Sprintf("filler%d", i), fmt.Sprintf("Unrelated task %d", i)}
	}
	return out
}

// sweepMux wires the team task endpoint to a per-page body table, recording
// every URL requested. Requests carrying a filter param (assignees[], list_ids[],
// space_ids[]) are served from filtered instead.
func sweepMux(tf *testutil.TestFactory, pages map[int]string, filtered string) *[]string {
	var urls []string
	var mu sync.Mutex
	tf.HandleFunc("team/12345/task", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		urls = append(urls, r.URL.String())
		mu.Unlock()

		q := r.URL.RawQuery
		body := `{"tasks":[]}`
		switch {
		case strings.Contains(q, "assignees%5B%5D=") || strings.Contains(q, "assignees[]=") ||
			strings.Contains(q, "list_ids") || strings.Contains(q, "space_ids"):
			body = filtered
		default:
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			if b, ok := pages[page]; ok {
				body = b
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(body))
	})
	return &urls
}

// A match on a later page must survive even when an earlier, narrower fetch
// already produced one. This is the 5.6.1 defect: the sprint/assignee tier hit
// first and returned, so the release card two pages in was never seen.
func TestSearch_ReturnsMatchesFromEveryPage(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	page0 := append(fillerPairs(99), [2]string{"A", "Tech debt 5.6.1 offline DB"})
	page1 := [][2]string{{"B", "iOS 5.6.1 Release Card"}}

	sweepMux(tf, map[int]string{
		0: searchTasksJSON(page0...),
		1: searchTasksJSON(page1...),
	}, searchTasksJSON([2]string{"A", "Tech debt 5.6.1 offline DB"}))

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Tech debt 5.6.1 offline DB", "page 0 match missing")
	assert.Contains(t, out, "iOS 5.6.1 Release Card", "page 1 match missing — sweep stopped early")
}

// Only an empty page ends the sweep. ClickUp does not return a reliably full
// page — a live page 0 comes back with 99 rows against a nominal size of 100 —
// so treating "shorter than a full page" as "workspace exhausted" stops the
// sweep dead on page 0 and hides everything behind it.
func TestSearch_ContinuesPastUnderFullPage(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	sweepMux(tf, map[int]string{
		0: searchTasksJSON(fillerPairs(99)...), // one short of nominal, as ClickUp does
		1: searchTasksJSON([2]string{"B", "Tech debt 5.6.1 profiling"}),
	}, `{"tasks":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.OutBuf.String(), "Tech debt 5.6.1 profiling",
		"sweep stopped on an under-full page")
}

// An empty page is the only trustworthy end-of-corpus signal; once seen, stop.
func TestSearch_StopsPaginatingOnEmptyPage(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	urls := sweepMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"A", "Only 5.6.1 card"}),
	}, `{"tasks":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, u := range *urls {
		assert.NotContains(t, u, "page=2", "kept paginating past an empty page")
	}
}

// When the page cap bites, say so. An absence of rows must never be readable
// as "there is nothing else".
func TestSearch_DisclosesTruncationAtPageCap(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	full := searchTasksJSON(append(fillerPairs(99), [2]string{"A", "5.6.1 card"})...)
	pages := map[int]string{}
	for i := 0; i < maxSweepPages+2; i++ {
		pages[i] = full
	}
	urls := sweepMux(tf, pages, `{"tasks":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.ErrBuf.String(), "Not shown:", "no truncation disclosure")

	var swept int
	for _, u := range *urls {
		if !strings.Contains(u, "assignees") {
			swept++
		}
	}
	assert.Equal(t, maxSweepPages, swept, "sweep should stop exactly at the cap")
}

// The `search=` param is dead weight: ClickUp accepts it and ignores it, so
// sending it invites the reader to believe filtering happened server-side.
func TestSearch_OmitsIgnoredServerSideSearchParam(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	urls := sweepMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"A", "Bug in the thing"}),
	}, `{"tasks":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "Bug", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, u := range *urls {
		assert.NotContains(t, u, "search=", "sent a search param ClickUp ignores")
	}
}

// The sweep only ever reads a prefix of the workspace (maxSweepPages), so the
// order that prefix arrives in decides what is reachable at all. `reverse=true`
// on order_by=updated means *oldest* first, which pointed the whole sweep at
// the least relevant end of the workspace: the tasks people are actually
// working on sit at the newest end. The old drill-down hid this because its
// narrow tiers returned fewer tasks than a page, so ordering never bit.
func TestSearch_SweepsNewestFirst(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	urls := sweepMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"A", "5.6.1 card"}),
	}, `{"tasks":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.NotEmpty(t, *urls)
	for _, u := range *urls {
		assert.Contains(t, u, "order_by=updated")
		assert.NotContains(t, u, "reverse=true", "reverse=true orders oldest-first")
	}
}
