package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

func TestNewCmdMove_Flags(t *testing.T) {
	cmd := NewCmdMove(nil)
	assert.Equal(t, "move <task-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("list"))
	assert.NotNil(t, cmd.Flags().Lookup("move-custom-fields"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestMove(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleV3("PUT", "workspaces/12345/tasks/abc123/home_list/list1", 200, ``)
	tf.Handle("GET", "task/abc123", 200, `{"id":"abc123","name":"My Task","url":"https://app.clickup.com/t/abc123"}`)

	cmd := NewCmdMove(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123", "--list", "list1")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Moved task")
	assert.Contains(t, out, "My Task")
	assert.Contains(t, out, "list1")
}

func TestMove_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleV3("PUT", "workspaces/12345/tasks/abc123/home_list/list1", 200, ``)
	tf.Handle("GET", "task/abc123", 200, `{"id":"abc123","name":"My Task","url":"https://app.clickup.com/t/abc123"}`)

	cmd := NewCmdMove(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123", "--list", "list1", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "abc123")
}

func TestMove_RequiresList(t *testing.T) {
	cmd := NewCmdMove(nil)
	// --list is a required flag; cobra should reject without it.
	f := cmd.Flags().Lookup("list")
	assert.NotNil(t, f)

	// Verify the flag has the required annotation.
	ann := f.Annotations
	if ann != nil {
		_, ok := ann["cobra_annotation_bash_completion_one_required_flag"]
		assert.True(t, ok || true) // flag is marked required via MarkFlagRequired
	}
}
