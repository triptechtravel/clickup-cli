package doc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdPageCreate_Flags(t *testing.T) {
	cmd := NewCmdPageCreate(nil)

	assert.Equal(t, "create <doc-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("parent-page-id"))
	assert.NotNil(t, cmd.Flags().Lookup("sub-title"))
	assert.NotNil(t, cmd.Flags().Lookup("content"))
	assert.NotNil(t, cmd.Flags().Lookup("content-format"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestNewCmdPageCreate_RequiresArg(t *testing.T) {
	cmd := NewCmdPageCreate(nil)

	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"doc-id"})
	assert.NoError(t, err)
}
