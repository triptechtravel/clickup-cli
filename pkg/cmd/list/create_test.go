package list

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

func TestNewCmdListCreate_Flags(t *testing.T) {
	cmd := NewCmdListCreate(nil)
	assert.Equal(t, "create", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("folder"))
	assert.NotNil(t, cmd.Flags().Lookup("space"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestListCreate_InFolder(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Mux.HandleFunc("/api/v2/folder/f1/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)
		assert.Equal(t, "Backlog", req["name"])

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"id": "l1", "name": "Backlog"}`))
	})

	cmd := NewCmdListCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "Backlog", "--folder", "f1")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Created list")
	assert.Contains(t, out, "Backlog")
	assert.Contains(t, out, "l1")
}

func TestListCreate_Folderless(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Mux.HandleFunc("/api/v2/space/s1/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)
		assert.Equal(t, "Backlog", req["name"])

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"id": "l2", "name": "Backlog"}`))
	})

	cmd := NewCmdListCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "Backlog", "--space", "s1")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Created list")
	assert.Contains(t, out, "Backlog")
	assert.Contains(t, out, "l2")
}

func TestListCreate_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Mux.HandleFunc("/api/v2/folder/f1/list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"id": "l1", "name": "Backlog"}`))
	})

	cmd := NewCmdListCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "Backlog", "--folder", "f1", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "l1", parsed["id"])
	assert.Equal(t, "Backlog", parsed["name"])
}

func TestListDelete(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("DELETE", "list/l1", 200, `{}`)

	cmd := NewCmdListDelete(tf.Factory)
	err := testutil.RunCommand(t, cmd, "l1", "--yes")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "deleted")
	assert.Contains(t, out, "l1")
}
