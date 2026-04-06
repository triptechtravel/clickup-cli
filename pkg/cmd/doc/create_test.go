package doc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdCreate_Flags(t *testing.T) {
	cmd := NewCmdCreate(nil)

	assert.Equal(t, "create", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("parent-id"))
	assert.NotNil(t, cmd.Flags().Lookup("parent-type"))
	assert.NotNil(t, cmd.Flags().Lookup("visibility"))
	assert.NotNil(t, cmd.Flags().Lookup("create-page"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestNewCmdCreate_VisibilityValidation(t *testing.T) {
	cases := []struct {
		vis     string
		wantErr bool
	}{
		{"PUBLIC", false},
		{"PRIVATE", false},
		{"PERSONAL", false},
		{"HIDDEN", false},
		{"invalid", true},
	}

	for _, tc := range cases {
		got := containsString(validVisibility, tc.vis)
		if tc.wantErr {
			assert.False(t, got, "expected invalid visibility %q to fail", tc.vis)
		} else {
			assert.True(t, got, "expected valid visibility %q to pass", tc.vis)
		}
	}
}

func TestNewCmdCreate_DefaultCreatePage(t *testing.T) {
	cmd := NewCmdCreate(nil)
	flag := cmd.Flags().Lookup("create-page")
	assert.NotNil(t, flag)
	assert.Equal(t, "true", flag.DefValue)
}
