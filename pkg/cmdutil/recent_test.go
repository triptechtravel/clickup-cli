package cmdutil

import (
	"strings"
	"testing"
)

func TestLocationSummary(t *testing.T) {
	tests := []struct {
		name     string
		tasks    []RecentTask
		expected []string
	}{
		{
			name:     "empty",
			tasks:    nil,
			expected: nil,
		},
		{
			name: "deduplicates same location",
			tasks: []RecentTask{
				{FolderName: "Sprint", ListName: "Sprint 84"},
				{FolderName: "Sprint", ListName: "Sprint 84"},
				{FolderName: "Backlog", ListName: "Bugs"},
			},
			expected: []string{"Sprint > Sprint 84", "Backlog > Bugs"},
		},
		{
			name: "folder only",
			tasks: []RecentTask{
				{FolderName: "Engineering"},
			},
			expected: []string{"Engineering"},
		},
		{
			name: "list only",
			tasks: []RecentTask{
				{ListName: "Sprint 84"},
			},
			expected: []string{"Sprint 84"},
		},
		{
			name: "multiple unique locations",
			tasks: []RecentTask{
				{FolderName: "Sprint Folder", ListName: "Sprint 84"},
				{FolderName: "Sprint Folder", ListName: "Sprint 83"},
				{FolderName: "Backlog", ListName: "Tech Debt"},
			},
			expected: []string{
				"Sprint Folder > Sprint 84",
				"Sprint Folder > Sprint 83",
				"Backlog > Tech Debt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LocationSummary(tt.tasks)
			if len(got) != len(tt.expected) {
				t.Errorf("LocationSummary() returned %d items, want %d: %v", len(got), len(tt.expected), got)
				return
			}
			for i, loc := range got {
				if loc != tt.expected[i] {
					t.Errorf("LocationSummary()[%d] = %q, want %q", i, loc, tt.expected[i])
				}
			}
		})
	}
}

func TestFormatRecentTaskOption(t *testing.T) {
	tests := []struct {
		name     string
		task     RecentTask
		contains []string
	}{
		{
			name: "with location",
			task: RecentTask{
				ID:         "abc123",
				Name:       "Fix the bug",
				Status:     "in progress",
				FolderName: "Sprint Folder",
				ListName:   "Sprint 84",
			},
			contains: []string{"[abc123]", "Fix the bug", "in progress", "Sprint Folder > Sprint 84"},
		},
		{
			name: "without location",
			task: RecentTask{
				ID:     "xyz789",
				Name:   "Some task",
				Status: "open",
			},
			contains: []string{"[xyz789]", "Some task", "open"},
		},
		{
			name: "folder only",
			task: RecentTask{
				ID:         "def456",
				Name:       "A task",
				Status:     "done",
				FolderName: "Backlog",
			},
			contains: []string{"[def456]", "A task", "in Backlog"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRecentTaskOption(tt.task)
			for _, s := range tt.contains {
				if !strings.Contains(got, s) {
					t.Errorf("FormatRecentTaskOption() = %q, want it to contain %q", got, s)
				}
			}
		})
	}
}
