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

func TestNewCmdSearch_CommentsFlag(t *testing.T) {
	cmd := NewCmdSearch(nil)
	assert.NotNil(t, cmd.Flags().Lookup("comments"))
}

func TestScoreTaskName_Substring(t *testing.T) {
	kind, rank, ok := scoreTaskName("geozone", "Geozone schema updates")
	assert.True(t, ok)
	assert.Equal(t, matchSubstring, kind)
	assert.Equal(t, 0, rank)
}

func TestScoreTaskName_NoMatch(t *testing.T) {
	_, _, ok := scoreTaskName("xyz123abc", "Totally different task")
	assert.False(t, ok)
}

func TestScoreTaskName_DescriptionMatch(t *testing.T) {
	tasks := []searchTask{
		{ID: "1", Name: "Unrelated task name", Description: "This task involves a geozone migration"},
	}

	matched, unmatched := filterTasks("geozone", tasks)
	assert.Len(t, matched, 1)
	assert.Equal(t, matchDescription, matched[0].kind)
	assert.Empty(t, unmatched)
}

func TestFilterTasks_NameBeatsDescription(t *testing.T) {
	tasks := []searchTask{
		{ID: "1", Name: "Geozone schema updates", Description: "This also mentions geozone"},
	}

	matched, _ := filterTasks("geozone", tasks)
	assert.Len(t, matched, 1)
	assert.Equal(t, matchSubstring, matched[0].kind) // name match takes priority
}

func TestFilterTasks_DescriptionFallback(t *testing.T) {
	tasks := []searchTask{
		{ID: "1", Name: "Update database schema", Description: "Migrate geozone tables to new format"},
		{ID: "2", Name: "Fix login bug", Description: "Users cannot log in properly"},
		{ID: "3", Name: "Geozone v2", Description: "New geozone implementation"},
	}

	matched, unmatched := filterTasks("geozone", tasks)

	// Task 1: description match, Task 3: name match, Task 2: no match
	assert.Len(t, matched, 2)
	assert.Len(t, unmatched, 1)
	assert.Equal(t, "2", unmatched[0].ID)

	// After sorting, name match should come first
	sortScoredTasks(matched)
	assert.Equal(t, "3", matched[0].ID) // name substring
	assert.Equal(t, matchSubstring, matched[0].kind)
	assert.Equal(t, "1", matched[1].ID) // description
	assert.Equal(t, matchDescription, matched[1].kind)
}
