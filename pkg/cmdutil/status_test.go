package cmdutil

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchStatus_ExactMatch(t *testing.T) {
	matched, err := MatchStatus("in progress", []string{"backlog", "in progress", "done"})
	require.NoError(t, err)
	assert.Equal(t, "in progress", matched)
}

func TestMatchStatus_CaseInsensitive(t *testing.T) {
	matched, err := MatchStatus("In Progress", []string{"backlog", "in progress", "done"})
	require.NoError(t, err)
	assert.Equal(t, "in progress", matched)
}

func TestMatchStatus_ContainsMatch(t *testing.T) {
	matched, err := MatchStatus("prog", []string{"backlog", "in progress", "done"})
	require.NoError(t, err)
	assert.Equal(t, "in progress", matched)
}

func TestMatchStatus_FuzzyMatch(t *testing.T) {
	matched, err := MatchStatus("progres", []string{"backlog", "in progress", "done"})
	require.NoError(t, err)
	assert.Equal(t, "in progress", matched)
}

func TestMatchStatus_NoMatch(t *testing.T) {
	_, err := MatchStatus("nonexistent", []string{"backlog", "in progress", "done"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching status found")
	assert.Contains(t, err.Error(), "Available statuses:")
}

func TestMatchStatus_ContainsPicksShortest(t *testing.T) {
	// "dev" matches both "in development" and "ready for development"
	// Should pick shortest (most specific).
	matched, err := MatchStatus("dev", []string{"in development", "ready for development", "done"})
	require.NoError(t, err)
	assert.Equal(t, "in development", matched)
}

func TestMatchAndReport_ExactNoWarning(t *testing.T) {
	var buf bytes.Buffer
	matched, err := matchAndReport("in progress", []string{"backlog", "in progress", "done"}, &buf)
	require.NoError(t, err)
	assert.Equal(t, "in progress", matched)
	assert.Empty(t, buf.String())
}

func TestMatchAndReport_FuzzyPrintsWarning(t *testing.T) {
	var buf bytes.Buffer
	matched, err := matchAndReport("prog", []string{"backlog", "in progress", "done"}, &buf)
	require.NoError(t, err)
	assert.Equal(t, "in progress", matched)
	assert.Contains(t, buf.String(), "matched to")
}

func TestMatchStatus_ListCustomStatuses(t *testing.T) {
	// Simulate list-level statuses (the bug scenario from issue #4).
	listStatuses := []string{"in analysis", "in development", "ready for development", "verification+"}
	matched, err := MatchStatus("in development", listStatuses)
	require.NoError(t, err)
	assert.Equal(t, "in development", matched)
}

func TestMatchStatus_ListCustomStatuses_NotInSpaceStatuses(t *testing.T) {
	// "in development" exists in list statuses but NOT in space statuses.
	spaceStatuses := []string{"backlog", "task is ready", "in-progress", "complete"}
	_, err := MatchStatus("in development", spaceStatuses)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching status found")
}
