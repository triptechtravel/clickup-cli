package space

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

func TestNewCmdSpaceCreate_Flags(t *testing.T) {
	cmd := NewCmdSpaceCreate(nil)
	assert.Equal(t, "create", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("team"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestSpaceCreate(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Mux.HandleFunc("/api/v2/team/12345/space", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)
		assert.Equal(t, "Dev", req["name"])

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"id": "s1", "name": "Dev", "private": false}`))
	})

	cmd := NewCmdSpaceCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "Dev")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Created space")
	assert.Contains(t, out, "Dev")
	assert.Contains(t, out, "s1")
}

func TestSpaceCreate_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Mux.HandleFunc("/api/v2/team/12345/space", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"id": "s1", "name": "Dev", "private": false}`))
	})

	cmd := NewCmdSpaceCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "Dev", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "s1", parsed["id"])
	assert.Equal(t, "Dev", parsed["name"])
}

func TestNewCmdSpaceDelete_Flags(t *testing.T) {
	cmd := NewCmdSpaceDelete(nil)
	assert.Equal(t, "delete <space-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("yes"))
	f := cmd.Flags().ShorthandLookup("y")
	assert.NotNil(t, f)
	assert.Equal(t, "yes", f.Name)
}

func TestSpaceDelete(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("DELETE", "space/s1", 200, `{}`)

	cmd := NewCmdSpaceDelete(tf.Factory)
	err := testutil.RunCommand(t, cmd, "s1", "--yes")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "deleted")
	assert.Contains(t, out, "s1")
}
