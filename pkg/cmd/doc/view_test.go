package doc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdView_Use(t *testing.T) {
	cmd := NewCmdView(nil)

	assert.Equal(t, "view <doc-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("jq"))
	assert.NotNil(t, cmd.Flags().Lookup("template"))
}

func TestNewCmdView_RequiresArg(t *testing.T) {
	cmd := NewCmdView(nil)

	// No args should fail validation
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	// Exactly one arg should pass
	err = cmd.Args(cmd, []string{"doc-id"})
	assert.NoError(t, err)
}
