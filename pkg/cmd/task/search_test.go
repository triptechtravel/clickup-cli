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

// ---------------------------------------------------------------------------
// sortScoredTasks
// ---------------------------------------------------------------------------

func TestSortScoredTasks_MixedKinds(t *testing.T) {
	tasks := []scoredTask{
		{searchTask: searchTask{ID: "1"}, kind: matchComment, fuzzyRank: 0},
		{searchTask: searchTask{ID: "2"}, kind: matchSubstring, fuzzyRank: 0},
		{searchTask: searchTask{ID: "3"}, kind: matchDescription, fuzzyRank: 0},
		{searchTask: searchTask{ID: "4"}, kind: matchFuzzy, fuzzyRank: 5},
	}

	sortScoredTasks(tasks)

	assert.Equal(t, matchSubstring, tasks[0].kind)
	assert.Equal(t, "2", tasks[0].ID)
	assert.Equal(t, matchFuzzy, tasks[1].kind)
	assert.Equal(t, "4", tasks[1].ID)
	assert.Equal(t, matchDescription, tasks[2].kind)
	assert.Equal(t, "3", tasks[2].ID)
	assert.Equal(t, matchComment, tasks[3].kind)
	assert.Equal(t, "1", tasks[3].ID)
}

func TestSortScoredTasks_SameKindFuzzyOrdering(t *testing.T) {
	tasks := []scoredTask{
		{searchTask: searchTask{ID: "1"}, kind: matchFuzzy, fuzzyRank: 10},
		{searchTask: searchTask{ID: "2"}, kind: matchFuzzy, fuzzyRank: 2},
		{searchTask: searchTask{ID: "3"}, kind: matchFuzzy, fuzzyRank: 5},
	}

	sortScoredTasks(tasks)

	assert.Equal(t, "2", tasks[0].ID) // rank 2 (best)
	assert.Equal(t, "3", tasks[1].ID) // rank 5
	assert.Equal(t, "1", tasks[2].ID) // rank 10 (worst)
}

// ---------------------------------------------------------------------------
// dedupScored
// ---------------------------------------------------------------------------

func TestDedupScored_KeepBestKind(t *testing.T) {
	tasks := []scoredTask{
		{searchTask: searchTask{ID: "1"}, kind: matchDescription, fuzzyRank: 0},
		{searchTask: searchTask{ID: "1"}, kind: matchSubstring, fuzzyRank: 0},
		{searchTask: searchTask{ID: "2"}, kind: matchComment, fuzzyRank: 0},
	}

	result := dedupScored(tasks)

	assert.Len(t, result, 2)
	// ID "1" should keep matchSubstring (lower kind = better)
	for _, r := range result {
		if r.ID == "1" {
			assert.Equal(t, matchSubstring, r.kind)
		}
	}
}

func TestDedupScored_KeepBestFuzzyRankForSameKind(t *testing.T) {
	tasks := []scoredTask{
		{searchTask: searchTask{ID: "1"}, kind: matchFuzzy, fuzzyRank: 10},
		{searchTask: searchTask{ID: "1"}, kind: matchFuzzy, fuzzyRank: 3},
	}

	result := dedupScored(tasks)

	assert.Len(t, result, 1)
	assert.Equal(t, 3, result[0].fuzzyRank) // keeps better rank
}

func TestDedupScored_PreservesOrder(t *testing.T) {
	tasks := []scoredTask{
		{searchTask: searchTask{ID: "3"}, kind: matchSubstring},
		{searchTask: searchTask{ID: "1"}, kind: matchSubstring},
		{searchTask: searchTask{ID: "2"}, kind: matchSubstring},
	}

	result := dedupScored(tasks)

	assert.Len(t, result, 3)
	assert.Equal(t, "3", result[0].ID)
	assert.Equal(t, "1", result[1].ID)
	assert.Equal(t, "2", result[2].ID)
}
