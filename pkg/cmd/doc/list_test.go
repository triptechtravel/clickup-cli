package doc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdList_Flags(t *testing.T) {
	cmd := NewCmdList(nil)

	assert.Equal(t, "list", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("creator"))
	assert.NotNil(t, cmd.Flags().Lookup("deleted"))
	assert.NotNil(t, cmd.Flags().Lookup("archived"))
	assert.NotNil(t, cmd.Flags().Lookup("parent-id"))
	assert.NotNil(t, cmd.Flags().Lookup("parent-type"))
	assert.NotNil(t, cmd.Flags().Lookup("limit"))
	assert.NotNil(t, cmd.Flags().Lookup("cursor"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("jq"))
	assert.NotNil(t, cmd.Flags().Lookup("template"))
}

func TestNewCmdList_ParentTypeValidation(t *testing.T) {
	cases := []struct {
		parentType string
		wantErr    bool
	}{
		{"SPACE", false},
		{"FOLDER", false},
		{"LIST", false},
		{"WORKSPACE", false},
		{"EVERYTHING", false},
		{"invalid", true},
	}

	for _, tc := range cases {
		_, err := parseParentType(tc.parentType)
		if tc.wantErr {
			assert.Error(t, err, "expected error for parent type %q", tc.parentType)
		} else {
			assert.NoError(t, err, "unexpected error for parent type %q", tc.parentType)
		}
	}
}
