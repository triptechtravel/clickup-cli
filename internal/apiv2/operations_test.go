package apiv2

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/api"
)

func testServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, api.NewTestClient(server.URL)
}

func TestUpdateTask_SendsPoints(t *testing.T) {
	var capturedBody map[string]any

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"task1","name":"Test","status":{"status":"open"}}`))
	})

	pts := float32(5)
	resp, err := UpdateTask(context.Background(), client, "task1", &clickupv2.UpdateTaskJSONRequest{
		Points: &pts,
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.EqualValues(t, 5, capturedBody["points"])
}

func TestUpdateTask_SendsMarkdownContent(t *testing.T) {
	var capturedBody map[string]any

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"task1","name":"Test","status":{"status":"open"}}`))
	})

	md := "# Hello\n\nWorld"
	_, err := UpdateTask(context.Background(), client, "task1", &clickupv2.UpdateTaskJSONRequest{
		MarkdownContent: &md,
	})

	require.NoError(t, err)
	assert.Equal(t, "# Hello\n\nWorld", capturedBody["markdown_content"])
}

// Regression test: the ClickUp API returns time_spent and time_estimate as
// numbers (milliseconds), but the generated UpdateTaskJSONResponse struct
// declares them as Nullable[string]. This causes json.Unmarshal to fail with:
//
//	json: cannot unmarshal number into Go struct field
//	    UpdateTaskJSONResponse.time_spent of type string
//
// This test reproduces the bug by returning a realistic API response with
// numeric time_spent and time_estimate fields.
func TestUpdateTask_NumericTimeSpent(t *testing.T) {
	// The ClickUp API returns time_spent and time_estimate as numbers
	// (milliseconds), not strings. This response includes those numeric
	// fields alongside other standard fields to reproduce the bug.
	const responseWithNumericTime = `{
		"id": "86d2ay3mz",
		"name": "Test task",
		"status": {"status": "in progress"},
		"time_estimate": 7200000,
		"time_spent": 3600000,
		"url": "https://app.clickup.com/t/86d2ay3mz"
	}`

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responseWithNumericTime))
	})

	md := "## Updated description"
	resp, err := UpdateTask(context.Background(), client, "86d2ay3mz", &clickupv2.UpdateTaskJSONRequest{
		MarkdownContent: &md,
	})

	require.NoError(t, err, "UpdateTask should not fail when API returns numeric time_spent/time_estimate")
	assert.NotNil(t, resp)
	assert.Equal(t, "86d2ay3mz", *resp.ID)
}

// Regression: the ClickUp API returns watchers, group_assignees, checklists,
// dependencies, and linked_tasks as arrays of objects, but the upstream OpenAPI
// spec declares them as string[]. patch-v2-spec.jq rewrites these to object[]
// before codegen; this test fails if that patch regresses, because
// json.Unmarshal on the UpdateTask response would error with:
//
//	json: cannot unmarshal object into Go struct field
//	    UpdateTaskJSONResponse.watchers of type string
//
// Surfaced by `clickup task edit --points N`, which calls UpdateTask directly
// (the points path bypasses the looser go-clickup wrapper).
func TestUpdateTask_ObjectArrayFields(t *testing.T) {
	const responseWithObjectArrays = `{
		"id": "86d2ay3mz",
		"name": "Test task",
		"status": {"status": "in progress"},
		"watchers": [
			{"id": 12345, "username": "alice", "email": "alice@example.com", "color": "#ff0000", "initials": "A", "profilePicture": null}
		],
		"group_assignees": [
			{"id": "group-1", "name": "Team A"}
		],
		"checklists": [
			{"id": "chk-1", "name": "QA", "items": []}
		],
		"dependencies": [
			{"task_id": "task-a", "depends_on": "task-b"}
		],
		"linked_tasks": [
			{"task_id": "task-c", "link_id": "link-1"}
		],
		"url": "https://app.clickup.com/t/86d2ay3mz"
	}`

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responseWithObjectArrays))
	})

	pts := float32(5)
	resp, err := UpdateTask(context.Background(), client, "86d2ay3mz", &clickupv2.UpdateTaskJSONRequest{
		Points: &pts,
	})

	require.NoError(t, err, "UpdateTask should decode responses where watchers/checklists/etc are object arrays")
	assert.NotNil(t, resp)
	assert.Equal(t, "86d2ay3mz", *resp.ID)
	assert.Len(t, resp.Watchers, 1, "watchers should decode as object array, not []string")
}

// Regression: same issue affects GetTask — the API returns numeric time_spent.
func TestGetTask_NumericTimeSpent(t *testing.T) {
	const response = `{
		"id": "task1",
		"name": "Task with time",
		"status": {"status": "open"},
		"time_estimate": 14400000,
		"time_spent": 7200000
	}`

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	})

	resp, err := GetTask(context.Background(), client, "task1")

	require.NoError(t, err, "GetTask should not fail when API returns numeric time_spent/time_estimate")
	assert.NotNil(t, resp)
}

func TestAddTagToTask_CorrectPath(t *testing.T) {
	var capturedPath string
	var capturedMethod string

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})

	_, err := AddTagToTask(context.Background(), client, "task1", "bug")
	require.NoError(t, err)
	assert.Equal(t, "POST", capturedMethod)
	assert.Equal(t, "/api/v2/task/task1/tag/bug", capturedPath)
}

func TestRemoveTagFromTask_CorrectPath(t *testing.T) {
	var capturedMethod string
	var capturedPath string

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})

	_, err := RemoveTagFromTask(context.Background(), client, "task1", "bug")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", capturedMethod)
	assert.Equal(t, "/api/v2/task/task1/tag/bug", capturedPath)
}

func TestCreateatimeentry_SendsRequest(t *testing.T) {
	var capturedBody map[string]any

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"id":"te1"}}`))
	})

	desc := "Working on task"
	tid := "task1"
	resp, err := Createatimeentry(context.Background(), client, "team1", &clickupv2.CreateatimeentryJSONRequest{
		Description: &desc,
		Start:       1700000000000,
		Duration:    3600000,
		Tid:         &tid,
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Working on task", capturedBody["description"])
	assert.EqualValues(t, 1700000000000, capturedBody["start"])
	assert.EqualValues(t, 3600000, capturedBody["duration"])
	assert.Equal(t, "task1", capturedBody["tid"])
}

func TestGetTaskComments_ReturnsComments(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"comments":[{"id":"c1","comment_text":"Hello"}]}`))
	})

	resp, err := GetTaskComments(context.Background(), client, "task1")
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestDeleteComment_CorrectPath(t *testing.T) {
	var capturedPath string
	var capturedMethod string

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})

	_, err := DeleteComment(context.Background(), client, "c123")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", capturedMethod)
	assert.Equal(t, "/api/v2/comment/c123", capturedPath)
}

func TestGetSpaceTags_ReturnsTypedData(t *testing.T) {
	var capturedPath string

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tags":[{"name":"bug","tag_fg":"#fff","tag_bg":"#f00"},{"name":"feature","tag_fg":"#000","tag_bg":"#0f0"}]}`))
	})

	resp, err := GetSpaceTags(context.Background(), client, "space1")
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "/api/v2/space/space1/tag", capturedPath)
	assert.Len(t, resp.Tags, 2)
	assert.Equal(t, "bug", resp.Tags[0].Name)
	assert.Equal(t, "#f00", resp.Tags[0].TagBg)
	assert.Equal(t, "feature", resp.Tags[1].Name)
}

func TestAddTaskToList_CorrectPath(t *testing.T) {
	var capturedPath string

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})

	_, err := AddTaskToList(context.Background(), client, "list1", "task1")
	require.NoError(t, err)
	assert.Equal(t, "/api/v2/list/list1/task/task1", capturedPath)
}

func TestApiError_ReturnsError(t *testing.T) {
	var callCount atomic.Int32

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(404)
		w.Write([]byte(`{"err":"Task not found","ECODE":"ITEM_015"}`))
	})

	_, err := GetTask(context.Background(), client, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}
