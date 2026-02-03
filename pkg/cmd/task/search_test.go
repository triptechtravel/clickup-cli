package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDedup(t *testing.T) {
	tasks := []searchTask{
		{ID: "1", Name: "Task A"},
		{ID: "2", Name: "Task B"},
		{ID: "1", Name: "Task A duplicate"},
		{ID: "3", Name: "Task C"},
		{ID: "2", Name: "Task B duplicate"},
	}

	result := dedup(tasks)
	assert.Len(t, result, 3)
	assert.Equal(t, "1", result[0].ID)
	assert.Equal(t, "Task A", result[0].Name) // keeps first occurrence
	assert.Equal(t, "2", result[1].ID)
	assert.Equal(t, "3", result[2].ID)
}

func TestDedup_Empty(t *testing.T) {
	result := dedup(nil)
	assert.Nil(t, result)
}

func TestNewCmdSearch_Flags(t *testing.T) {
	cmd := NewCmdSearch(nil)

	assert.NotNil(t, cmd.Flags().Lookup("space"))
	assert.NotNil(t, cmd.Flags().Lookup("folder"))
	assert.NotNil(t, cmd.Flags().Lookup("pick"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.Equal(t, "search <query>", cmd.Use)
}
