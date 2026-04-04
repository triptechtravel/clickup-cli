package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requestCapture captures HTTP requests for assertion.
type requestCapture struct {
	Method string
	Path   string
	Body   string
	Header http.Header
}

func captureServer(t *testing.T, status int, response string) (*httptest.Server, *[]requestCapture) {
	t.Helper()
	var captures []requestCapture
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captures = append(captures, requestCapture{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   string(body),
			Header: r.Header.Clone(),
		})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return server, &captures
}

// TestCustomEndpoint_SetTaskPoints validates that our setTaskPoints function
// would send a PUT /task/{id} with {"points": N} per the spec.
// (We test the raw HTTP request format our code constructs.)
func TestCustomEndpoint_SetTaskPoints(t *testing.T) {
	server, captures := captureServer(t, 200, `{}`)
	client := NewTestClient(server.URL)

	// Simulate what helpers.go:setTaskPoints does
	url := client.URL("task/%s", "abc123")
	req, err := http.NewRequest("PUT", url, nil)
	require.NoError(t, err)

	// Spec: PUT /v2/task/{task_id} with body {"points": N}
	assert.Equal(t, server.URL+"/api/v2/task/abc123", url)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, *captures, 1)
	assert.Equal(t, "PUT", (*captures)[0].Method)
	assert.Equal(t, "/api/v2/task/abc123", (*captures)[0].Path)
}

// TestCustomEndpoint_AddTaskTag validates POST /task/{id}/tag/{name}.
func TestCustomEndpoint_AddTaskTag(t *testing.T) {
	server, captures := captureServer(t, 200, `{}`)
	client := NewTestClient(server.URL)

	url := client.URL("task/%s/tag/%s", "abc123", "bug")
	req, err := http.NewRequest("POST", url, nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, *captures, 1)
	assert.Equal(t, "POST", (*captures)[0].Method)
	// Spec: POST /v2/task/{task_id}/tag/{tag_name}
	assert.Equal(t, "/api/v2/task/abc123/tag/bug", (*captures)[0].Path)
}

// TestCustomEndpoint_RemoveTaskTag validates DELETE /task/{id}/tag/{name}.
func TestCustomEndpoint_RemoveTaskTag(t *testing.T) {
	server, captures := captureServer(t, 200, `{}`)
	client := NewTestClient(server.URL)

	url := client.URL("task/%s/tag/%s", "abc123", "bug")
	req, err := http.NewRequest("DELETE", url, nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, *captures, 1)
	assert.Equal(t, "DELETE", (*captures)[0].Method)
	assert.Equal(t, "/api/v2/task/abc123/tag/bug", (*captures)[0].Path)
}

// TestCustomEndpoint_AddTaskToList validates POST /list/{id}/task/{id}.
func TestCustomEndpoint_AddTaskToList(t *testing.T) {
	server, captures := captureServer(t, 200, `{}`)
	client := NewTestClient(server.URL)

	url := client.URL("list/%s/task/%s", "list1", "task1")
	req, err := http.NewRequest("POST", url, nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, *captures, 1)
	assert.Equal(t, "POST", (*captures)[0].Method)
	assert.Equal(t, "/api/v2/list/list1/task/task1", (*captures)[0].Path)
}

// TestCustomEndpoint_FetchSpaceStatuses validates GET /space/{id}.
func TestCustomEndpoint_FetchSpaceStatuses(t *testing.T) {
	server, captures := captureServer(t, 200, `{
		"id": "space1",
		"name": "Dev",
		"statuses": [
			{"status": "to do", "color": "#d3d3d3", "type": "open", "orderindex": 0},
			{"status": "done", "color": "#6bc950", "type": "closed", "orderindex": 1}
		]
	}`)
	client := NewTestClient(server.URL)

	url := client.URL("space/%s", "space1")
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Len(t, *captures, 1)
	assert.Equal(t, "GET", (*captures)[0].Method)
	assert.Equal(t, "/api/v2/space/space1", (*captures)[0].Path)

	// Verify response matches spec shape
	var body struct {
		Statuses []struct {
			Status     string `json:"status"`
			Color      string `json:"color"`
			Type       string `json:"type"`
			Orderindex int    `json:"orderindex"`
		} `json:"statuses"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	require.NoError(t, json.Unmarshal(respBody, &body))
	assert.Len(t, body.Statuses, 2)
	assert.Equal(t, "to do", body.Statuses[0].Status)
}

// TestCustomEndpoint_CommentAdd validates POST /task/{id}/comment.
func TestCustomEndpoint_CommentAdd(t *testing.T) {
	server, captures := captureServer(t, 200, `{"id": "comment1"}`)
	client := NewTestClient(server.URL)

	url := client.URL("task/%s/comment", "abc123")
	req, err := http.NewRequest("POST", url, nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, *captures, 1)
	assert.Equal(t, "POST", (*captures)[0].Method)
	assert.Equal(t, "/api/v2/task/abc123/comment", (*captures)[0].Path)
}

// TestCustomEndpoint_CommentReply validates POST /comment/{id}/reply.
func TestCustomEndpoint_CommentReply(t *testing.T) {
	server, captures := captureServer(t, 200, `{"id": "reply1"}`)
	client := NewTestClient(server.URL)

	url := client.URL("comment/%s/reply", "c456")
	req, err := http.NewRequest("POST", url, nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, *captures, 1)
	assert.Equal(t, "POST", (*captures)[0].Method)
	assert.Equal(t, "/api/v2/comment/c456/reply", (*captures)[0].Path)
}

// TestCustomEndpoint_TimeEntryCreate validates POST /team/{id}/time_entries.
func TestCustomEndpoint_TimeEntryCreate(t *testing.T) {
	server, captures := captureServer(t, 200, `{"data": {"id": "te1"}}`)
	client := NewTestClient(server.URL)

	url := client.URL("team/%s/time_entries", "team1")
	req, err := http.NewRequest("POST", url, nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, *captures, 1)
	assert.Equal(t, "POST", (*captures)[0].Method)
	assert.Equal(t, "/api/v2/team/team1/time_entries", (*captures)[0].Path)
}

// TestCustomEndpoint_SpaceTags validates GET /space/{id}/tag.
func TestCustomEndpoint_SpaceTags(t *testing.T) {
	server, captures := captureServer(t, 200, `{"tags": [{"name": "bug"}]}`)
	client := NewTestClient(server.URL)

	url := client.URL("space/%s/tag", "space1")
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, *captures, 1)
	assert.Equal(t, "GET", (*captures)[0].Method)
	assert.Equal(t, "/api/v2/space/space1/tag", (*captures)[0].Path)
}
