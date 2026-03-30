package comment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdReply_Usage(t *testing.T) {
	cmd := NewCmdReply(nil)
	assert.Equal(t, "reply <comment-id> [BODY]", cmd.Use)
}

func TestNewCmdReply_RequiresArgs(t *testing.T) {
	cmd := NewCmdReply(nil)
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.NoError(t, cmd.Args(cmd, []string{"123"}))
	assert.NoError(t, cmd.Args(cmd, []string{"123", "body"}))
	assert.Error(t, cmd.Args(cmd, []string{"123", "body", "extra"}))
}

func TestNewCmdReply_EditorFlag(t *testing.T) {
	cmd := NewCmdReply(nil)
	assert.NotNil(t, cmd.Flags().Lookup("editor"))
}
