package list

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/config"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

var sampleListsJSON = `{
	"lists": [
		{"id": "l1", "name": "To Do", "task_count": "3"},
		{"id": "l2", "name": "In Progress", "task_count": "7"}
	]
}`

func listsHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(body))
	}
}

func TestListList_WithFolder(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("folder/f1/list", listsHandler(sampleListsJSON))

	cmd := NewCmdListList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--folder", "f1")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "l1")
	assert.Contains(t, out, "To Do")
	assert.Contains(t, out, "l2")
	assert.Contains(t, out, "In Progress")
}

func TestListList_WithSpace(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("space/67890/list", listsHandler(sampleListsJSON))

	// Clear folder so it falls through to space (folderless)
	tf.Factory.SetConfig(&config.Config{
		Workspace: "12345",
		Space:     "67890",
	})

	cmd := NewCmdListList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "l1")
	assert.Contains(t, out, "To Do")
}

func TestListList_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("folder/f1/list", listsHandler(sampleListsJSON))

	cmd := NewCmdListList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--folder", "f1", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed), "output should be valid JSON")
	assert.Len(t, parsed, 2)
	assert.Equal(t, "l1", parsed[0]["id"])
	assert.Equal(t, "To Do", parsed[0]["name"])
}
