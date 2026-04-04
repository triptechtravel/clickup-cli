package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raksul/go-clickup/clickup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoClickup_GetTask_RequestFormat verifies that go-clickup sends the
// correct request format for GetTask as defined by the V2 spec.
func TestGoClickup_GetTask_RequestFormat(t *testing.T) {
	var captured *http.Request
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Clone(r.Context())
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		// Minimal valid task response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "abc123",
			"name": "Test task",
			"status": map[string]interface{}{
				"status": "open",
			},
		})
	}))
	defer server.Close()

	client := NewTestClient(server.URL)

	// Spec: GET /v2/task/{task_id}
	// Optional query params: custom_task_ids, team_id, include_subtasks, include_markdown_description
	opts := &clickup.GetTaskOptions{
		CustomTaskIDs:   true,
		TeamID:          12345,
		IncludeSubTasks: true,
	}
	task, _, err := client.Clickup.Tasks.GetTask(context.Background(), "abc123", opts)

	require.NoError(t, err)
	assert.Equal(t, "abc123", task.ID)
	assert.Equal(t, "GET", captured.Method)
	assert.Contains(t, captured.URL.Path, "/task/abc123")
	assert.Empty(t, capturedBody, "GET request should have no body")

	// Verify query params match spec
	q := captured.URL.Query()
	assert.Equal(t, "true", q.Get("custom_task_ids"))
	assert.Equal(t, "12345", q.Get("team_id"))
	assert.Equal(t, "true", q.Get("include_subtasks"))
}

// TestGoClickup_CreateTask_RequestFormat verifies CreateTask sends correct
// request body per the V2 spec.
func TestGoClickup_CreateTask_RequestFormat(t *testing.T) {
	var captured *http.Request
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Clone(r.Context())
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "new123",
			"name": "New task",
			"status": map[string]interface{}{
				"status": "to do",
			},
		})
	}))
	defer server.Close()

	client := NewTestClient(server.URL)

	req := &clickup.TaskRequest{
		Name:        "New task",
		Description: "A description",
		Priority:    2,
	}
	task, _, err := client.Clickup.Tasks.CreateTask(context.Background(), "list123", req)

	require.NoError(t, err)
	assert.Equal(t, "new123", task.ID)
	assert.Equal(t, "POST", captured.Method)
	assert.Contains(t, captured.URL.Path, "/list/list123/task")

	// Verify request body matches spec schema
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))
	assert.Equal(t, "New task", body["name"])
	assert.Equal(t, "A description", body["description"])
	assert.EqualValues(t, 2, body["priority"])
}

// TestGoClickup_UpdateTask_MissingPoints verifies that go-clickup's
// TaskUpdateRequest does NOT support `points` — confirming our raw HTTP
// workaround is needed. This is a known gap.
func TestGoClickup_UpdateTask_MissingPoints(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "abc123",
			"name": "Test task",
			"status": map[string]interface{}{
				"status": "open",
			},
		})
	}))
	defer server.Close()

	client := NewTestClient(server.URL)

	// go-clickup TaskUpdateRequest has no Points field
	req := &clickup.TaskUpdateRequest{
		Name: "Updated name",
	}
	_, _, err := client.Clickup.Tasks.UpdateTask(context.Background(), "abc123", nil, req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))

	// Confirm points is NOT in the request body — go-clickup can't send it
	_, hasPoints := body["points"]
	assert.False(t, hasPoints, "go-clickup TaskUpdateRequest should NOT support points (known gap)")

	// Confirm markdown_description is also missing
	_, hasMD := body["markdown_content"]
	assert.False(t, hasMD, "go-clickup TaskUpdateRequest should NOT support markdown_content (known gap)")
}

// TestGoClickup_UpdateTask_BodyFormat verifies the fields go-clickup DOES send.
func TestGoClickup_UpdateTask_BodyFormat(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "abc123",
			"name": "Updated",
			"status": map[string]interface{}{
				"status": "in progress",
			},
		})
	}))
	defer server.Close()

	client := NewTestClient(server.URL)

	req := &clickup.TaskUpdateRequest{
		Name:   "Updated",
		Status: "in progress",
	}
	_, _, err := client.Clickup.Tasks.UpdateTask(context.Background(), "abc123", nil, req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))
	assert.Equal(t, "Updated", body["name"])
	assert.Equal(t, "in progress", body["status"])
}

// TestGoClickup_GetSpaces_RequestFormat verifies GetSpaces request.
func TestGoClickup_GetSpaces_RequestFormat(t *testing.T) {
	var captured *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"spaces": []map[string]interface{}{
				{"id": "space1", "name": "Dev"},
			},
		})
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	spaces, _, err := client.Clickup.Spaces.GetSpaces(context.Background(), "team123", false)

	require.NoError(t, err)
	assert.Len(t, spaces, 1)
	assert.Equal(t, "GET", captured.Method)
	assert.Contains(t, captured.URL.Path, "/team/team123/space")
	assert.Equal(t, "false", captured.URL.Query().Get("archived"))
}

// TestGoClickup_TaskResponse_NullableFields verifies go-clickup handles
// nullable fields from the spec (custom_id, priority, due_date can be null).
func TestGoClickup_TaskResponse_NullableFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Spec says custom_id, priority, due_date can be null
		w.Write([]byte(`{
			"id": "abc123",
			"custom_id": null,
			"name": "Test task",
			"status": {"status": "open", "color": "#fff"},
			"priority": null,
			"due_date": null,
			"start_date": null,
			"points": 5.0,
			"assignees": [],
			"tags": []
		}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	task, _, err := client.Clickup.Tasks.GetTask(context.Background(), "abc123", nil)

	require.NoError(t, err)
	assert.Equal(t, "abc123", task.ID)
	assert.Equal(t, "Test task", task.Name)
	// Check nullable fields don't cause errors
	assert.Empty(t, task.CustomID)
	// BUG: go-clickup does not handle nullable priority correctly.
	// Spec says priority can be null, but go-clickup always initializes
	// the struct (TaskPriority{Priority:"", Color:""}). Our CLI code
	// must check Priority.Priority == "" to detect null.
	assert.Empty(t, task.Priority.Priority, "null priority should have empty Priority string")
}

// TestGoClickup_GetList_StatusesParsed verifies go-clickup parses list statuses.
func TestGoClickup_GetList_StatusesParsed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "list1",
			"name": "Sprint 1",
			"statuses": [
				{"status": "to do", "color": "#d3d3d3", "type": "open", "orderindex": 0},
				{"status": "in progress", "color": "#4194f6", "type": "custom", "orderindex": 1},
				{"status": "done", "color": "#6bc950", "type": "closed", "orderindex": 2}
			]
		}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	list, _, err := client.Clickup.Lists.GetList(context.Background(), "list1")

	require.NoError(t, err)
	assert.Equal(t, "list1", list.ID)
	assert.Len(t, list.Statuses, 3, "go-clickup should parse list statuses")
	assert.Equal(t, "to do", list.Statuses[0].Status)
	assert.Equal(t, "done", list.Statuses[2].Status)
}

// TestAuthTransport_InjectsHeaders verifies our authTransport adds the
// correct headers per the V2 spec's security requirements.
func TestAuthTransport_InjectsHeaders(t *testing.T) {
	var captured *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.DoRequest(req)

	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "test-token", captured.Header.Get("Authorization"))
	assert.True(t, strings.HasPrefix(captured.Header.Get("User-Agent"), "clickup-cli/"))
}

// TestAuthTransport_RetryOn429 verifies the single retry on rate limit.
func TestAuthTransport_RetryOn429(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "0")
			w.WriteHeader(429)
			w.Write([]byte(`{"err": "Rate limit exceeded"}`))
			return
		}
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.DoRequest(req)

	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 2, attempts, "should retry once on 429")
}

// TestAuthTransport_429ThenAnother429 verifies no infinite retry.
func TestAuthTransport_429ThenAnother429(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "0")
		w.WriteHeader(429)
		w.Write([]byte(`{"err": "Rate limit exceeded"}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.DoRequest(req)

	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 429, resp.StatusCode, "should return 429 after retry fails")
	assert.Equal(t, 2, attempts, "should only retry once")
}

// TestAuthTransport_401WithoutECODE returns AuthExpiredError.
func TestAuthTransport_401WithoutECODE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(401)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	_, err := client.DoRequest(req)

	require.Error(t, err)
	var authErr *AuthExpiredError
	assert.ErrorAs(t, err, &authErr, "401 without ECODE should be AuthExpiredError")
}

// TestAuthTransport_401WithECODE returns APIError (permission, not auth expiry).
func TestAuthTransport_401WithECODE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(401)
		w.Write([]byte(`{"err": "Token invalid", "ECODE": "OAUTH_025"}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	_, err := client.DoRequest(req)

	require.Error(t, err)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "401 with ECODE should be APIError, not AuthExpiredError")
	assert.Equal(t, 401, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "ECODE")
}

// TestAuthTransport_401EmptyBody returns AuthExpiredError.
func TestAuthTransport_401EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(401)
		// Empty body
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	_, err := client.DoRequest(req)

	require.Error(t, err)
	var authErr *AuthExpiredError
	assert.ErrorAs(t, err, &authErr, "401 with empty body should be AuthExpiredError")
}

// TestClient_URL verifies the URL helper builds correct paths.
func TestClient_URL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	client := NewTestClient(server.URL)

	assert.Equal(t, server.URL+"/api/v2/task/abc123", client.URL("task/%s", "abc123"))
	assert.Equal(t, server.URL+"/api/v2/task/abc123/tag/bug", client.URL("task/%s/tag/%s", "abc123", "bug"))
	assert.Equal(t, server.URL+"/api/v2/space/123", client.URL("space/%s", "123"))
}
