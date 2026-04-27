package task

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
func teamsJSON(members ...struct{ ID int; Username string }) string {
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

func makeMember(id int, username string) struct{ ID int; Username string } {
	return struct{ ID int; Username string }{ID: id, Username: username}
}

func setupTeamAndUser(tf *testutil.TestFactory, currentUserID int, members ...struct{ ID int; Username string }) {
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
// Server-side search test
// ---------------------------------------------------------------------------

func TestSearchServerSide(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// Mock the team tasks endpoint — verify search= param is passed.
	var capturedURL string
	tf.HandleFunc("team/12345/task", func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"tasks":[{"id":"abc","name":"Server Bug Fix","status":{"status":"open"},"assignees":[]}]}`))
	})

	// User endpoint for Level 2 drill-down (won't reach it since Level 0 succeeds).
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	cmd := NewCmdSearch(tf.Factory)
	err := testutil.RunCommand(t, cmd, "Bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify search= was passed to the API.
	assert.True(t, strings.Contains(capturedURL, "search=Bug"),
		"expected search=Bug in URL, got: %s", capturedURL)

	// Verify the task appeared in output.
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
