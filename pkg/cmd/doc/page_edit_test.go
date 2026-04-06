package doc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdPageEdit_Flags(t *testing.T) {
	cmd := NewCmdPageEdit(nil)

	assert.Equal(t, "edit <doc-id> <page-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("sub-title"))
	assert.NotNil(t, cmd.Flags().Lookup("content"))
	assert.NotNil(t, cmd.Flags().Lookup("content-format"))
	assert.NotNil(t, cmd.Flags().Lookup("content-edit-mode"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestNewCmdPageEdit_DefaultEditMode(t *testing.T) {
	cmd := NewCmdPageEdit(nil)
	flag := cmd.Flags().Lookup("content-edit-mode")
	assert.NotNil(t, flag)
	assert.Equal(t, "replace", flag.DefValue)
}

func TestNewCmdPageEdit_RequiresExactlyTwoArgs(t *testing.T) {
	cmd := NewCmdPageEdit(nil)

	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"doc-id"})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"doc-id", "page-id"})
	assert.NoError(t, err)
}

func TestNewCmdPageEdit_EditModeValidation(t *testing.T) {
	cases := []struct {
		mode    string
		isValid bool
	}{
		{"replace", true},
		{"append", true},
		{"prepend", true},
		{"delete", false},
		{"overwrite", false},
	}

	for _, tc := range cases {
		got := containsString(validEditModes, tc.mode)
		assert.Equal(t, tc.isValid, got, "edit mode %q", tc.mode)
	}
}
