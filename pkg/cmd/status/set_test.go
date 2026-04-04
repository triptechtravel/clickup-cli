package status

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

const taskJSON = `{
	"id": "task1",
	"name": "Test task",
	"status": {"status": "to do"},
	"space": {"id": "space1"},
	"list": {"id": "list1"},
	"priority": null,
	"assignees": [],
	"tags": []
}`

const listWithStatusesJSON = `{
	"id": "list1",
	"name": "Sprint",
	"statuses": [
		{"status": "to do", "color": "#d3d3d3", "type": "open", "orderindex": 0},
		{"status": "in progress", "color": "#4194f6", "type": "custom", "orderindex": 1},
		{"status": "done", "color": "#6bc950", "type": "closed", "orderindex": 2}
	],
	"space": {"id": "space1"}
}`

const spaceStatusesJSON = `{
	"id": "space1",
	"statuses": [
		{"status": "open", "color": "#d3d3d3"},
		{"status": "closed", "color": "#6bc950"}
	]
}`

// registerTaskHandlers sets up the GET /task/{id} handlers (with and without trailing slash).
func registerTaskHandlers(tf *testutil.TestFactory, taskBody string) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(taskBody))
	}
	tf.HandleFunc("task/task1", handler)
	tf.HandleFunc("task/task1/", handler)
}

// registerListHandler sets up the GET /list/{id} handler.
func registerListHandler(tf *testutil.TestFactory, listBody string) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(listBody))
	}
	tf.HandleFunc("list/list1", handler)
	tf.HandleFunc("list/list1/", handler)
}

// registerSpaceHandler sets up the GET /space/{id} handler.
func registerSpaceHandler(tf *testutil.TestFactory, spaceBody string) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(spaceBody))
	}
	tf.HandleFunc("space/space1", handler)
	tf.HandleFunc("space/space1/", handler)
}

// registerUpdateHandler sets up the PUT /task/{id} handler and captures the request body.
func registerUpdateHandler(tf *testutil.TestFactory, capturedBody *string) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		*capturedBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"task1","status":{"status":"done"}}`))
	}
	tf.HandleFunc("task/task1", handler)
	tf.HandleFunc("task/task1/", handler)
}

func TestStatusSet_ExactMatch(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody string

	// The task endpoint serves both GET (fetch task) and PUT (update status).
	tf.HandleFunc("task/task1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(taskJSON))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"task1","status":{"status":"done"}}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	tf.HandleFunc("task/task1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(taskJSON))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"task1","status":{"status":"done"}}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	registerListHandler(tf, listWithStatusesJSON)

	cmd := NewCmdSet(tf.Factory)
	err := testutil.RunCommand(t, cmd, "done", "task1")
	require.NoError(t, err)

	// Verify the PUT payload contains the exact matched status.
	require.NotEmpty(t, capturedBody, "expected PUT request to be sent")
	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(capturedBody), &payload))
	assert.Equal(t, "done", payload["status"])

	// Verify output contains success message.
	out := tf.OutBuf.String()
	assert.Contains(t, out, "Status changed")
}

func TestStatusSet_FuzzyMatch(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody string

	tf.HandleFunc("task/task1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(taskJSON))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"task1","status":{"status":"in progress"}}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	tf.HandleFunc("task/task1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(taskJSON))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"task1","status":{"status":"in progress"}}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	registerListHandler(tf, listWithStatusesJSON)

	cmd := NewCmdSet(tf.Factory)
	err := testutil.RunCommand(t, cmd, "prog", "task1")
	require.NoError(t, err)

	// Verify the PUT payload contains the fuzzy-matched status.
	require.NotEmpty(t, capturedBody, "expected PUT request to be sent")
	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(capturedBody), &payload))
	assert.Equal(t, "in progress", payload["status"])

	// Verify stderr contains the fuzzy match notice.
	errOut := tf.ErrBuf.String()
	assert.Contains(t, errOut, "matched to")
}

func TestStatusSet_NoMatch(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	registerTaskHandlers(tf, taskJSON)

	// List with no statuses forces fallback to space statuses.
	listNoStatuses := `{
		"id": "list1",
		"name": "Sprint",
		"statuses": [],
		"space": {"id": "space1"}
	}`
	registerListHandler(tf, listNoStatuses)
	registerSpaceHandler(tf, spaceStatusesJSON)

	cmd := NewCmdSet(tf.Factory)
	err := testutil.RunCommand(t, cmd, "xyz", "task1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no matching status found")
}

func TestStatusSet_ListLevelStatuses(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody string

	// Custom list statuses that differ from space statuses.
	customListJSON := `{
		"id": "list1",
		"name": "Sprint",
		"statuses": [
			{"status": "backlog", "color": "#d3d3d3", "type": "open", "orderindex": 0},
			{"status": "in review", "color": "#4194f6", "type": "custom", "orderindex": 1},
			{"status": "shipped", "color": "#6bc950", "type": "closed", "orderindex": 2}
		],
		"space": {"id": "space1"}
	}`

	tf.HandleFunc("task/task1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(taskJSON))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"task1","status":{"status":"shipped"}}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	tf.HandleFunc("task/task1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(taskJSON))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"task1","status":{"status":"shipped"}}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	registerListHandler(tf, customListJSON)
	// Register space handler too -- but it should NOT be called since list has statuses.
	registerSpaceHandler(tf, spaceStatusesJSON)

	cmd := NewCmdSet(tf.Factory)
	err := testutil.RunCommand(t, cmd, "shipped", "task1")
	require.NoError(t, err)

	// Verify the PUT payload uses the list-level status, not space-level.
	require.NotEmpty(t, capturedBody, "expected PUT request to be sent")
	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(capturedBody), &payload))
	assert.Equal(t, "shipped", payload["status"])

	// "shipped" is a list-level status, not available at space level.
	// If space statuses were used instead, this would have failed or matched something else.
	out := tf.OutBuf.String()
	assert.Contains(t, out, "Status changed")
}
