package folder

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

func TestNewCmdFolderCreate_Flags(t *testing.T) {
	cmd := NewCmdFolderCreate(nil)
	assert.Equal(t, "create", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("space"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestFolderCreate(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Mux.HandleFunc("/api/v2/space/67890/folder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)
		assert.Equal(t, "Sprint Folder", req["name"])

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"id": "f1", "name": "Sprint Folder", "orderindex": 0}`))
	})

	cmd := NewCmdFolderCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "Sprint Folder")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Created folder")
	assert.Contains(t, out, "Sprint Folder")
	assert.Contains(t, out, "f1")
}

func TestFolderCreate_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Mux.HandleFunc("/api/v2/space/67890/folder", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"id": "f1", "name": "Sprint Folder", "orderindex": 0}`))
	})

	cmd := NewCmdFolderCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "Sprint Folder", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "f1", parsed["id"])
	assert.Equal(t, "Sprint Folder", parsed["name"])
}

func TestFolderDelete(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("DELETE", "folder/f1", 200, `{}`)

	cmd := NewCmdFolderDelete(tf.Factory)
	err := testutil.RunCommand(t, cmd, "f1", "--yes")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "deleted")
	assert.Contains(t, out, "f1")
}
