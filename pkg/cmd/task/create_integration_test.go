package task

import (
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

const createdTaskJSON = `{
	"id": "new123",
	"custom_id": null,
	"name": "Test task",
	"status": {"status": "to do", "color": "#d3d3d3"},
	"priority": null,
	"creator": {"id": 1, "username": "test"},
	"assignees": [],
	"tags": [],
	"url": "https://app.clickup.com/t/new123"
}`

func TestCreateCommand_WithNameAndListID(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// CreateTask POSTs to list/{id}/task (no trailing slash)
	tf.Handle("POST", "list/list1/task", 200, createdTaskJSON)

	cmd := NewCmdCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--list-id", "list1", "--name", "Test task")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Test task")
	assert.Contains(t, out, "new123")
}

func TestCreateCommand_WithTags(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// CreateTask POST
	tf.Handle("POST", "list/list1/task", 200, createdTaskJSON)

	// Tag endpoint — addTaskTag uses raw HTTP POST to task/{id}/tag/{name}
	var tagCalled atomic.Int32
	tagHandler := func(w http.ResponseWriter, r *http.Request) {
		tagCalled.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}
	tf.HandleFunc("task/new123/tag/bug", tagHandler)

	// Also register the list endpoint for tag validation (GetList)
	// go-clickup GetList uses list/{id}/ with trailing slash
	listJSON := `{"id": "list1", "name": "Test List", "space": {"id": ""}, "statuses": []}`
	tf.HandleFunc("list/list1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(listJSON))
	})
	tf.HandleFunc("list/list1/", func(w http.ResponseWriter, r *http.Request) {
		// Only handle GET (for GetList), let POST fall through to CreateTask
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Remaining", "99")
			w.Write([]byte(listJSON))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	cmd := NewCmdCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--list-id", "list1", "--name", "Test task", "--tags", "bug")
	require.NoError(t, err)

	assert.Equal(t, int32(1), tagCalled.Load(), "tag endpoint should have been called once")

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Test task")
}

func TestCreateCommand_WithPoints(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// CreateTask POST
	tf.Handle("POST", "list/list1/task", 200, createdTaskJSON)

	// setTaskPoints uses raw HTTP PUT to task/{id}
	var pointsCalled atomic.Int32
	pointsHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		pointsCalled.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(createdTaskJSON))
	}
	tf.HandleFunc("task/new123", pointsHandler)
	tf.HandleFunc("task/new123/", pointsHandler)

	cmd := NewCmdCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--list-id", "list1", "--name", "Test task", "--points", "5")
	require.NoError(t, err)

	assert.Equal(t, int32(1), pointsCalled.Load(), "points endpoint should have been called once")

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Test task")
}

func TestCreateCommand_MissingListID(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	cmd := NewCmdCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "Test task")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either --list-id or --current is required")
}

func TestCreateCommand_NoNameNonInteractive(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// IOStreams.IsTerminal() returns false by default in test factory
	// (In is io.NopCloser, not a real terminal)

	// CreateTask POST (registered but should not be called)
	tf.Handle("POST", "list/list1/task", 200, createdTaskJSON)

	cmd := NewCmdCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--list-id", "list1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}
