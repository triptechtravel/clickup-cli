package doc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdPageView_Flags(t *testing.T) {
	cmd := NewCmdPageView(nil)

	assert.Equal(t, "view <doc-id> <page-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("content-format"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestNewCmdPageView_RequiresExactlyTwoArgs(t *testing.T) {
	cmd := NewCmdPageView(nil)

	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"doc-id"})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"doc-id", "page-id"})
	assert.NoError(t, err)
}

func TestNewCmdPageView_ContentFormatValidation(t *testing.T) {
	cases := []struct {
		format  string
		isValid bool
	}{
		{"text/md", true},
		{"text/plain", true},
		{"html", false},
	}

	for _, tc := range cases {
		got := containsString(validContentFormats, tc.format)
		assert.Equal(t, tc.isValid, got, "format %q", tc.format)
	}
}
