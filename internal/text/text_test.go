package text

import (
	"testing"
	"time"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxWidth int
		want     string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"hello", 5, "hello"},
		{"hello", 4, "h..."},
	}

	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxWidth)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
		}
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		count    int
		singular string
		want     string
	}{
		{0, "task", "0 tasks"},
		{1, "task", "1 task"},
		{5, "task", "5 tasks"},
	}

	for _, tt := range tests {
		got := Pluralize(tt.count, tt.singular)
		if got != tt.want {
			t.Errorf("Pluralize(%d, %q) = %q, want %q", tt.count, tt.singular, got, tt.want)
		}
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		input time.Time
		want  string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5 minutes ago"},
		{now.Add(-2 * time.Hour), "2 hours ago"},
		{now.Add(-48 * time.Hour), "2 days ago"},
	}

	for _, tt := range tests {
		got := RelativeTime(tt.input)
		if got != tt.want {
			t.Errorf("RelativeTime(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPriorityName(t *testing.T) {
	tests := []struct {
		priority int
		want     string
	}{
		{1, "Urgent"},
		{2, "High"},
		{3, "Normal"},
		{4, "Low"},
		{0, "None"},
	}

	for _, tt := range tests {
		got := PriorityName(tt.priority)
		if got != tt.want {
			t.Errorf("PriorityName(%d) = %q, want %q", tt.priority, got, tt.want)
		}
	}
}
