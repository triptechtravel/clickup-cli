package task

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

var sampleTaskJSON = `{
	"id": "abc123",
	"custom_id": null,
	"name": "Fix login bug",
	"text_content": "The login form fails on Safari",
	"description": "The login form fails on Safari",
	"markdown_description": "The login form fails on **Safari**",
	"status": {"status": "in progress", "color": "#4194f6", "type": "custom"},
	"priority": {"priority": "high", "color": "#f50000"},
	"creator": {"id": 1, "username": "isaac", "email": "isaac@test.com"},
	"assignees": [{"id": 1, "username": "isaac"}],
	"tags": [{"name": "bug"}],
	"url": "https://app.clickup.com/t/abc123",
	"date_created": "1700000000000",
	"date_updated": "1700100000000",
	"due_date": null,
	"points": 3,
	"time_estimate": null,
	"subtasks": []
}`

func taskHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(body))
	}
}

func TestViewCommand_ByID(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// go-clickup sends both /task/{id} and /task/{id}/ (with trailing slash)
	tf.HandleFunc("task/abc123", taskHandler(sampleTaskJSON))
	tf.HandleFunc("task/abc123/", taskHandler(sampleTaskJSON))

	cmd := NewCmdView(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Fix login bug")
	assert.Contains(t, out, "in progress")
	assert.Contains(t, out, "abc123")
}

func TestViewCommand_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleFunc("task/abc123", taskHandler(sampleTaskJSON))
	tf.HandleFunc("task/abc123/", taskHandler(sampleTaskJSON))

	cmd := NewCmdView(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	// Should be valid JSON
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed), "output should be valid JSON")
	assert.Equal(t, "abc123", parsed["id"])
	assert.Equal(t, "Fix login bug", parsed["name"])
}

func TestViewCommand_NotFound(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.Handle("GET", "task/nonexistent", 404, `{"err": "Task not found", "ECODE": "ITEM_015"}`)

	cmd := NewCmdView(tf.Factory)
	err := testutil.RunCommand(t, cmd, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch task")
}

func TestViewCommand_WithSubtasks(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	taskWithSubtasks := `{
		"id": "parent1",
		"name": "Parent task",
		"status": {"status": "open", "color": "#d3d3d3"},
		"priority": null,
		"creator": {"id": 1, "username": "isaac"},
		"assignees": [],
		"tags": [],
		"url": "https://app.clickup.com/t/parent1",
		"date_created": "1700000000000",
		"date_updated": "1700100000000",
		"subtasks": [
			{"id": "sub1", "name": "Subtask 1", "status": {"status": "done"}, "assignees": []},
			{"id": "sub2", "name": "Subtask 2", "status": {"status": "open"}, "assignees": []}
		]
	}`

	tf.HandleFunc("task/parent1", taskHandler(taskWithSubtasks))
	tf.HandleFunc("task/parent1/", taskHandler(taskWithSubtasks))

	cmd := NewCmdView(tf.Factory)
	err := testutil.RunCommand(t, cmd, "parent1")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Parent task")
	assert.Contains(t, out, "Subtask 1")
	assert.Contains(t, out, "Subtask 2")
}
