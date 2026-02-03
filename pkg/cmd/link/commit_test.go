package link

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveCommitSHA(t *testing.T) {
	// HEAD should always resolve in a git repo.
	sha, err := resolveCommitSHA("HEAD")
	assert.NoError(t, err)
	assert.Len(t, sha, 40, "full SHA should be 40 hex characters")
}

func TestGetCommitMessage(t *testing.T) {
	msg, err := getCommitMessage("HEAD")
	assert.NoError(t, err)
	assert.NotEmpty(t, msg)
}

func TestNewCmdLinkCommit_Flags(t *testing.T) {
	cmd := NewCmdLinkCommit(nil)

	// Verify flags exist.
	assert.NotNil(t, cmd.Flags().Lookup("task"))
	assert.NotNil(t, cmd.Flags().Lookup("repo"))
}

func TestNewCmdLinkCommit_Usage(t *testing.T) {
	cmd := NewCmdLinkCommit(nil)
	assert.Equal(t, "commit [SHA]", cmd.Use)
}
