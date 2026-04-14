package template

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

var sampleTaskTemplatesJSON = `{
	"templates": ["template-1", "template-2"]
}`

var sampleFolderTemplatesJSON = `{
	"templates": [
		{"name": "Sprint Folder", "id": "t-100"},
		{"name": "Project Folder", "id": "t-200"}
	]
}`

var sampleListTemplatesJSON = `{
	"templates": [
		{"name": "Bug Tracker", "id": "t-300"},
		{"name": "Feature Board", "id": "t-400"}
	]
}`

func templatesHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(body))
	}
}

func TestTemplateList_Task(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/taskTemplate", templatesHandler(sampleTaskTemplatesJSON))

	cmd := NewCmdTemplateList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "template-1")
	assert.Contains(t, out, "template-2")
}

func TestTemplateList_Folder(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/folder_template", templatesHandler(sampleFolderTemplatesJSON))

	cmd := NewCmdTemplateList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--type", "folder")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Sprint Folder")
	assert.Contains(t, out, "t-100")
	assert.Contains(t, out, "Project Folder")
	assert.Contains(t, out, "t-200")
}

func TestTemplateList_List(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/list_template", templatesHandler(sampleListTemplatesJSON))

	cmd := NewCmdTemplateList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--type", "list")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Bug Tracker")
	assert.Contains(t, out, "t-300")
}

func TestTemplateUse(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}
	tf.HandleFunc("list/list-99/taskTemplate/t-12345", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	})

	cmd := NewCmdTemplateUse(tf.Factory)
	err := testutil.RunCommand(t, cmd, "t-12345", "--list", "list-99", "--name", "New Bug")
	require.NoError(t, err)

	assert.Equal(t, "New Bug", capturedBody["name"])

	out := tf.OutBuf.String()
	assert.Contains(t, out, "New Bug")
	assert.Contains(t, out, "t-12345")
}

func TestTemplateList_Empty(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/taskTemplate", templatesHandler(`{"templates": []}`))

	cmd := NewCmdTemplateList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "No task templates found.")
}
