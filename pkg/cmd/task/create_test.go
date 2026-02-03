package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdCreate_Flags(t *testing.T) {
	cmd := NewCmdCreate(nil)

	assert.NotNil(t, cmd.Flags().Lookup("list-id"))
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("description"))
	assert.NotNil(t, cmd.Flags().Lookup("status"))
	assert.NotNil(t, cmd.Flags().Lookup("priority"))
	assert.NotNil(t, cmd.Flags().Lookup("assignee"))
	assert.NotNil(t, cmd.Flags().Lookup("tags"))
	assert.NotNil(t, cmd.Flags().Lookup("due-date"))
	assert.NotNil(t, cmd.Flags().Lookup("start-date"))
	assert.NotNil(t, cmd.Flags().Lookup("time-estimate"))
	assert.NotNil(t, cmd.Flags().Lookup("points"))
}
