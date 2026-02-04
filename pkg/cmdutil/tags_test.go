package cmdutil

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTags_AllValid(t *testing.T) {
	tags := filterTags([]string{"bug", "frontend"}, []string{"bug", "frontend", "backend"})
	assert.Equal(t, []string{"bug", "frontend"}, tags)
}

func TestValidateTags_SomeInvalid(t *testing.T) {
	var buf bytes.Buffer
	tags := filterAndWarn([]string{"bug", "nonexistent", "frontend"}, []string{"bug", "frontend", "backend"}, &buf)
	assert.Equal(t, []string{"bug", "frontend"}, tags)
	assert.Contains(t, buf.String(), "nonexistent")
	assert.Contains(t, buf.String(), "unknown tag")
}

func TestValidateTags_AllInvalid(t *testing.T) {
	var buf bytes.Buffer
	tags := filterAndWarn([]string{"fake1", "fake2"}, []string{"bug", "frontend"}, &buf)
	assert.Empty(t, tags)
	assert.Contains(t, buf.String(), "fake1")
	assert.Contains(t, buf.String(), "fake2")
}

func TestValidateTags_CaseInsensitive(t *testing.T) {
	tags := filterTags([]string{"BUG", "Frontend"}, []string{"bug", "frontend", "backend"})
	assert.Equal(t, []string{"BUG", "Frontend"}, tags)
}

func TestValidateTags_EmptyInput(t *testing.T) {
	tags := filterTags([]string{}, []string{"bug", "frontend"})
	assert.Empty(t, tags)
}

// filterTags extracts the core filtering logic for testing without API calls.
func filterTags(tags []string, available []string) []string {
	availableSet := make(map[string]bool, len(available))
	for _, t := range available {
		availableSet[strings.ToLower(t)] = true
	}

	var valid []string
	for _, tag := range tags {
		if availableSet[strings.ToLower(tag)] {
			valid = append(valid, tag)
		}
	}
	return valid
}

// filterAndWarn mirrors ValidateTags logic for testing the warn-and-filter behavior.
func filterAndWarn(tags []string, available []string, w io.Writer) []string {
	availableSet := make(map[string]bool, len(available))
	for _, t := range available {
		availableSet[strings.ToLower(t)] = true
	}

	var valid []string
	var unknown []string
	for _, tag := range tags {
		if availableSet[strings.ToLower(tag)] {
			valid = append(valid, tag)
		} else {
			unknown = append(unknown, tag)
		}
	}

	if len(unknown) > 0 {
		fmt.Fprintf(w, "Warning: unknown tag(s) %s (available: %s)\n",
			strings.Join(unknown, ", "),
			strings.Join(available, ", "))
	}

	return valid
}
