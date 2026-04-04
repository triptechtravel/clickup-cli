package task

import (
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

const deleteTaskJSON = `{
	"id": "del456",
	"custom_id": null,
	"name": "Task to delete",
	"status": {"status": "to do", "color": "#d3d3d3"},
	"priority": null,
	"creator": {"id": 1, "username": "test"},
	"assignees": [],
	"tags": [],
	"url": "https://app.clickup.com/t/del456"
}`

func TestDeleteCommand_WithYesFlag(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// Both GetTask and DeleteTask use task/{id}/ — register a single handler
	// that dispatches on HTTP method.
	var deleteCalled atomic.Int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")

		switch r.Method {
		case "GET":
			w.Write([]byte(deleteTaskJSON))
		case "DELETE":
			deleteCalled.Add(1)
			w.WriteHeader(200)
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
	tf.HandleFunc("task/del456", handler)
	tf.HandleFunc("task/del456/", handler)

	cmd := NewCmdDelete(tf.Factory)
	err := testutil.RunCommand(t, cmd, "del456", "--yes")
	require.NoError(t, err)

	assert.Equal(t, int32(1), deleteCalled.Load(), "DELETE endpoint should have been called once")

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Task to delete")
	assert.Contains(t, out, "deleted")
}

func TestDeleteCommand_NonInteractiveProceeds(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// Non-interactive mode (IOStreams.IsTerminal() = false in test factory)
	// With --yes flag, delete should proceed without prompting

	var getCalled atomic.Int32
	var deleteCalled atomic.Int32

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")

		switch r.Method {
		case "GET":
			getCalled.Add(1)
			w.Write([]byte(deleteTaskJSON))
		case "DELETE":
			deleteCalled.Add(1)
			w.WriteHeader(200)
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}

	tf.HandleFunc("task/del456", handler)
	tf.HandleFunc("task/del456/", handler)

	cmd := NewCmdDelete(tf.Factory)
	err := testutil.RunCommand(t, cmd, "del456", "--yes")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, getCalled.Load(), int32(1), "GET should have been called to fetch task info")
	assert.Equal(t, int32(1), deleteCalled.Load(), "DELETE should have been called")

	out := tf.OutBuf.String()
	assert.Contains(t, out, "deleted")
}
