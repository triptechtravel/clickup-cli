package folder

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

var sampleFoldersJSON = `{
	"folders": [
		{"id": "f1", "name": "Sprint Backlog", "task_count": "5", "archived": false},
		{"id": "f2", "name": "Product Roadmap", "task_count": "12", "archived": false}
	]
}`

func foldersHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(body))
	}
}

func TestFolderList(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("space/67890/folder", foldersHandler(sampleFoldersJSON))

	cmd := NewCmdFolderList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "f1")
	assert.Contains(t, out, "Sprint Backlog")
	assert.Contains(t, out, "f2")
	assert.Contains(t, out, "Product Roadmap")
}

func TestFolderList_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("space/67890/folder", foldersHandler(sampleFoldersJSON))

	cmd := NewCmdFolderList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed), "output should be valid JSON")
	assert.Len(t, parsed, 2)
	assert.Equal(t, "f1", parsed[0]["id"])
	assert.Equal(t, "Sprint Backlog", parsed[0]["name"])
}

func TestFolderList_Empty(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("space/67890/folder", foldersHandler(`{"folders": []}`))

	cmd := NewCmdFolderList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "No folders found.")
}
