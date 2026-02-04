package git

import (
	"regexp"
	"strings"
)

var (
	// CU-{alphanumeric} pattern for default ClickUp task IDs
	cuIDPattern = regexp.MustCompile(`(?i)CU-([0-9a-z]+)`)

	// PREFIX-{number} pattern for custom task IDs (e.g., PROJ-42, ENG-1234)
	customIDPattern = regexp.MustCompile(`([A-Z][A-Z0-9]+-\d+)`)

	// Git branch prefixes to strip before matching
	branchPrefixes = []string{
		"feature/", "fix/", "hotfix/", "bugfix/", "release/",
		"chore/", "docs/", "refactor/", "test/", "ci/",
	}

	// Uppercase prefixes to exclude from custom ID matching
	excludedPrefixes = map[string]bool{
		"FEATURE": true, "BUGFIX": true, "RELEASE": true,
		"HOTFIX": true, "FIX": true, "CHORE": true,
		"DOCS": true, "REFACTOR": true, "TEST": true,
	}
)

// TaskIDResult holds a parsed task ID from a branch name.
type TaskIDResult struct {
	// Raw is the full matched string (e.g., "CU-ae27de" or "PROJ-42")
	Raw string
	// ID is the task identifier suitable for API calls
	ID string
	// IsCustomID indicates whether this is a custom prefix ID vs a CU- ID
	IsCustomID bool
}

// ExtractTaskID attempts to find a ClickUp task ID in a branch name.
// It tries CU-{hex} first, then PREFIX-{number} patterns.
func ExtractTaskID(branch string) *TaskIDResult {
	cleaned := stripBranchPrefix(branch)

	// Try CU-{alphanumeric} pattern first
	if matches := cuIDPattern.FindStringSubmatch(cleaned); len(matches) >= 2 {
		return &TaskIDResult{
			Raw:        matches[0],
			ID:         matches[1], // Strip CU- prefix; API expects raw ID
			IsCustomID: false,
		}
	}

	// Try custom PREFIX-NUMBER pattern
	if matches := customIDPattern.FindStringSubmatch(cleaned); len(matches) >= 2 {
		prefix := strings.Split(matches[1], "-")[0]
		if excludedPrefixes[prefix] {
			return nil
		}
		return &TaskIDResult{
			Raw:        matches[1],
			ID:         matches[1],
			IsCustomID: true,
		}
	}

	return nil
}

func stripBranchPrefix(branch string) string {
	for _, prefix := range branchPrefixes {
		if strings.HasPrefix(branch, prefix) {
			return strings.TrimPrefix(branch, prefix)
		}
	}
	return branch
}

// ParseTaskID normalizes a task ID string passed as a CLI argument.
// It handles CU- prefixed IDs (stripping the prefix), custom prefix IDs
// (e.g., PROJ-42), and raw IDs (returned as-is).
func ParseTaskID(input string) *TaskIDResult {
	// Check for CU- prefix (default ClickUp ID format)
	if matches := cuIDPattern.FindStringSubmatch(input); len(matches) >= 2 {
		return &TaskIDResult{
			Raw:        matches[0],
			ID:         matches[1], // Strip CU- prefix; API expects raw ID
			IsCustomID: false,
		}
	}

	// Check for custom PREFIX-NUMBER pattern
	if matches := customIDPattern.FindStringSubmatch(input); len(matches) >= 2 {
		prefix := strings.Split(matches[1], "-")[0]
		if !excludedPrefixes[prefix] {
			return &TaskIDResult{
				Raw:        matches[1],
				ID:         matches[1],
				IsCustomID: true,
			}
		}
	}

	// Assume it's a raw ClickUp task ID
	return &TaskIDResult{
		Raw:        input,
		ID:         input,
		IsCustomID: false,
	}
}

// BranchNamingSuggestion returns a suggestion message for when no task ID is found.
func BranchNamingSuggestion(branch string) string {
	return "No ClickUp task ID found in branch \"" + branch + "\".\n\n" +
		"Tip: Include the task ID in your branch name for auto-detection:\n" +
		"  git checkout -b CU-abc123-my-feature-branch\n" +
		"  git checkout -b PROJ-42-my-feature-branch\n"
}
