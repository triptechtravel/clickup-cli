package view

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

var sampleViewsJSON = `{
	"views": [
		{"id": "v1", "name": "Board View", "type": "board"},
		{"id": "v2", "name": "List View", "type": "list"}
	]
}`

var sampleViewTasksJSON = `{
	"tasks": [
		{"id": "task1", "name": "Fix bug"},
		{"id": "task2", "name": "Add feature"}
	],
	"last_page": true
}`

func viewsHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(body))
	}
}

func TestViewList_Team(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/view", viewsHandler(sampleViewsJSON))

	cmd := NewCmdViewList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--team")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "v1")
	assert.Contains(t, out, "Board View")
	assert.Contains(t, out, "v2")
	assert.Contains(t, out, "List View")
}

func TestViewList_Space(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("space/67890/view", viewsHandler(sampleViewsJSON))

	cmd := NewCmdViewList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--space", "67890")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Board View")
}

func TestViewList_Empty(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/view", viewsHandler(`{"views": []}`))

	cmd := NewCmdViewList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "No views found")
}

func TestViewTasks(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("view/v1/task", viewsHandler(sampleViewTasksJSON))

	cmd := NewCmdViewTasks(tf.Factory)
	err := testutil.RunCommand(t, cmd, "v1")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "task1")
	assert.Contains(t, out, "Fix bug")
	assert.Contains(t, out, "task2")
	assert.Contains(t, out, "Add feature")
}

func TestViewGet(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "view/v1", 200, `{
		"view": {
			"id": "v1",
			"name": "Board View",
			"type": "board",
			"parent": {"id":"12345","type":7}
		}
	}`)

	cmd := NewCmdViewGet(tf.Factory)
	err := testutil.RunCommand(t, cmd, "v1")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed), "output should be valid JSON")
	view := parsed["view"].(map[string]interface{})
	assert.Equal(t, "v1", view["id"])
	assert.Equal(t, "Board View", view["name"])
}

func TestViewTasks_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("view/v1/task", viewsHandler(sampleViewTasksJSON))

	cmd := NewCmdViewTasks(tf.Factory)
	err := testutil.RunCommand(t, cmd, "v1", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed), "output should be valid JSON")
	assert.Len(t, parsed, 2)
	assert.Equal(t, "task1", parsed[0]["id"])
}
