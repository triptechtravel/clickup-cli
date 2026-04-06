package doc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdPageList_Flags(t *testing.T) {
	cmd := NewCmdPageList(nil)

	assert.Equal(t, "list <doc-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("max-depth"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("jq"))
	assert.NotNil(t, cmd.Flags().Lookup("template"))
}

func TestNewCmdPageList_DefaultMaxDepth(t *testing.T) {
	cmd := NewCmdPageList(nil)
	flag := cmd.Flags().Lookup("max-depth")
	assert.Equal(t, "-1", flag.DefValue)
}

func TestNewCmdPageList_RequiresArg(t *testing.T) {
	cmd := NewCmdPageList(nil)

	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"doc-id"})
	assert.NoError(t, err)
}
