package attachment

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

func TestAddCommand_SingleFile(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// Create a temp file to upload.
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("hello world"), 0644))

	// Mock the v2 attachment upload endpoint.
	tf.HandleFunc("task/abc123/attachment", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{
			"id": "att_new",
			"title": "test.txt",
			"extension": "txt",
			"url": "https://attachments.clickup.com/test.txt"
		}`))
	})

	cmd := NewCmdAdd(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123", tmpFile)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Uploaded")
	assert.Contains(t, out, "test.txt")
}

func TestAddCommand_FileAutoDetect(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// Create a temp file.
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "photo.png")
	require.NoError(t, os.WriteFile(tmpFile, []byte("png data"), 0644))

	// Set git context so task is auto-detected.
	tf.Factory.SetGitContext(&git.RepoContext{
		Branch: "feature/CU-abc123-my-task",
		TaskID: &git.TaskIDResult{
			Raw: "CU-abc123",
			ID:  "abc123",
		},
	})

	tf.HandleFunc("task/abc123/attachment", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{
			"id": "att_new",
			"title": "photo.png",
			"extension": "png",
			"url": "https://attachments.clickup.com/photo.png"
		}`))
	})

	cmd := NewCmdAdd(tf.Factory)
	err := testutil.RunCommand(t, cmd, tmpFile)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Uploaded")
	assert.Contains(t, out, "photo.png")
	assert.Contains(t, tf.ErrBuf.String(), "Detected task")
}

func TestAddCommand_NoArgs(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	cmd := NewCmdAdd(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	assert.Error(t, err)
}

func TestAddCommand_MultipleFiles(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "a.txt")
	file2 := filepath.Join(tmpDir, "b.txt")
	require.NoError(t, os.WriteFile(file1, []byte("aaa"), 0644))
	require.NoError(t, os.WriteFile(file2, []byte("bbb"), 0644))

	uploadCount := 0
	tf.HandleFunc("task/abc123/attachment", func(w http.ResponseWriter, r *http.Request) {
		uploadCount++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id": "att", "title": "file", "url": "https://example.com/file"}`))
	})

	cmd := NewCmdAdd(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123", file1, file2)
	require.NoError(t, err)
	assert.Equal(t, 2, uploadCount)
}
