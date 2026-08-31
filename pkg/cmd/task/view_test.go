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

func TestViewCommand_RecursiveSubtasks(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	parentJSON := `{
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

	// sub1 has a grandchild; sub2 has no children.
	sub1JSON := `{
		"id": "sub1",
		"name": "Subtask 1",
		"status": {"status": "done"},
		"subtasks": [
			{"id": "grand1", "name": "Grandchild 1", "status": {"status": "open"}, "assignees": []}
		]
	}`
	sub2JSON := `{
		"id": "sub2",
		"name": "Subtask 2",
		"status": {"status": "open"},
		"subtasks": []
	}`
	grand1JSON := `{
		"id": "grand1",
		"name": "Grandchild 1",
		"status": {"status": "open"},
		"subtasks": []
	}`

	tf.HandleFunc("task/parent1", taskHandler(parentJSON))
	tf.HandleFunc("task/parent1/", taskHandler(parentJSON))
	tf.HandleFunc("task/sub1/", taskHandler(sub1JSON))
	tf.HandleFunc("task/sub2/", taskHandler(sub2JSON))
	tf.HandleFunc("task/grand1/", taskHandler(grand1JSON))

	cmd := NewCmdView(tf.Factory)
	err := testutil.RunCommand(t, cmd, "parent1", "--recursive", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))

	// Check subtasks exist.
	subtasks, ok := parsed["subtasks"].([]interface{})
	require.True(t, ok)
	require.Len(t, subtasks, 2)

	// sub1 should have a nested subtask (grandchild).
	sub1 := subtasks[0].(map[string]interface{})
	assert.Equal(t, "sub1", sub1["id"])
	grandchildren, ok := sub1["subtasks"].([]interface{})
	require.True(t, ok)
	require.Len(t, grandchildren, 1)
	assert.Equal(t, "grand1", grandchildren[0].(map[string]interface{})["id"])

	// sub2 should have no nested subtasks (empty or nil).
	sub2 := subtasks[1].(map[string]interface{})
	assert.Equal(t, "sub2", sub2["id"])
}

// Issue #27: one subtask with logged time made the whole parent unreadable —
// name, status and every sibling included — because ClickUp sends time_spent as
// a string once time is tracked and as a number when it is not.
func TestViewCommand_SubtaskWithTrackedTime(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	taskJSON := `{
		"id": "86cay82n2",
		"name": "Parent task",
		"status": {"status": "in progress", "color": "#4194f6"},
		"creator": {"id": 1, "username": "isaac"},
		"assignees": [],
		"tags": [],
		"url": "https://app.clickup.com/t/86cay82n2",
		"date_created": "1700000000000",
		"date_updated": "1700100000000",
		"time_spent": 0,
		"subtasks": [
			{"id": "86caxfujb", "name": "Untracked subtask", "status": {"status": "open"}, "assignees": [], "time_spent": 0},
			{"id": "86caygxwe", "name": "Tracked subtask", "status": {"status": "done"}, "assignees": [], "time_spent": "2040000"}
		]
	}`

	tf.HandleFunc("task/86cay82n2", taskHandler(taskJSON))
	tf.HandleFunc("task/86cay82n2/", taskHandler(taskJSON))

	cmd := NewCmdView(tf.Factory)
	require.NoError(t, testutil.RunCommand(t, cmd, "86cay82n2"))

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Parent task")
	assert.Contains(t, out, "Untracked subtask")
	assert.Contains(t, out, "Tracked subtask")
}

// The same task through --json: time_spent stays a JSON number whichever form
// the API sent, so jq filters over the output keep working.
func TestViewCommand_JSONNormalisesTrackedTime(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	taskJSON := `{
		"id": "86caygxwe",
		"name": "Tracked task",
		"status": {"status": "done"},
		"creator": {"id": 1, "username": "isaac"},
		"assignees": [],
		"tags": [],
		"date_created": "1700000000000",
		"date_updated": "1700100000000",
		"time_spent": "2040000",
		"time_estimate": "3600000",
		"subtasks": []
	}`

	tf.HandleFunc("task/86caygxwe", taskHandler(taskJSON))
	tf.HandleFunc("task/86caygxwe/", taskHandler(taskJSON))

	cmd := NewCmdView(tf.Factory)
	require.NoError(t, testutil.RunCommand(t, cmd, "86caygxwe", "--json"))

	var parsed struct {
		TimeSpent    json.Number `json:"time_spent"`
		TimeEstimate json.Number `json:"time_estimate"`
	}
	require.NoError(t, json.Unmarshal(tf.OutBuf.Bytes(), &parsed))
	assert.Equal(t, "2040000", parsed.TimeSpent.String())
	assert.Equal(t, "3600000", parsed.TimeEstimate.String())
}
