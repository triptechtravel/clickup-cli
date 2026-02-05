package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDate(t *testing.T) {
	d, err := parseDate("2025-06-15")
	assert.NoError(t, err)
	assert.NotNil(t, d)

	_, err = parseDate("invalid")
	assert.Error(t, err)

	_, err = parseDate("15-06-2025")
	assert.Error(t, err)
}

func TestParseDuration(t *testing.T) {
	ms, err := parseDuration("2h")
	assert.NoError(t, err)
	assert.Equal(t, 7200000, ms)

	ms, err = parseDuration("30m")
	assert.NoError(t, err)
	assert.Equal(t, 1800000, ms)

	ms, err = parseDuration("1h30m")
	assert.NoError(t, err)
	assert.Equal(t, 5400000, ms)

	_, err = parseDuration("invalid")
	assert.Error(t, err)
}

func TestFormatMillisDuration(t *testing.T) {
	assert.Equal(t, "2h", formatMillisDuration(7200000))
	assert.Equal(t, "30m", formatMillisDuration(1800000))
	assert.Equal(t, "1h 30m", formatMillisDuration(5400000))
	assert.Equal(t, "", formatMillisDuration(0))
	assert.Equal(t, "", formatMillisDuration(-1))
	assert.Equal(t, "< 1m", formatMillisDuration(500))
}

func TestDiffTags(t *testing.T) {
	tests := []struct {
		name       string
		current    []string
		desired    []string
		wantAdd    []string
		wantRemove []string
	}{
		{
			name:       "no changes when identical",
			current:    []string{"a", "b"},
			desired:    []string{"a", "b"},
			wantAdd:    nil,
			wantRemove: nil,
		},
		{
			name:       "add new tags from empty",
			current:    nil,
			desired:    []string{"fix", "urgent"},
			wantAdd:    []string{"fix", "urgent"},
			wantRemove: nil,
		},
		{
			name:       "remove all tags",
			current:    []string{"fix", "urgent"},
			desired:    nil,
			wantAdd:    nil,
			wantRemove: []string{"fix", "urgent"},
		},
		{
			name:       "add and remove with overlap",
			current:    []string{"a", "b"},
			desired:    []string{"b", "c"},
			wantAdd:    []string{"c"},
			wantRemove: []string{"a"},
		},
		{
			name:       "add only",
			current:    []string{"a"},
			desired:    []string{"a", "b", "c"},
			wantAdd:    []string{"b", "c"},
			wantRemove: nil,
		},
		{
			name:       "remove only",
			current:    []string{"a", "b", "c"},
			desired:    []string{"a"},
			wantAdd:    nil,
			wantRemove: []string{"b", "c"},
		},
		{
			name:       "both empty",
			current:    nil,
			desired:    nil,
			wantAdd:    nil,
			wantRemove: nil,
		},
		{
			name:       "complete replacement",
			current:    []string{"old1", "old2"},
			desired:    []string{"new1", "new2"},
			wantAdd:    []string{"new1", "new2"},
			wantRemove: []string{"old1", "old2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdd, gotRemove := diffTags(tt.current, tt.desired)
			assert.Equal(t, tt.wantAdd, gotAdd)
			assert.Equal(t, tt.wantRemove, gotRemove)
		})
	}
}
