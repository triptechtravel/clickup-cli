package text

import (
	"fmt"
	"strings"
	"time"
)

// Truncate shortens a string to maxWidth, adding "..." if truncated.
func Truncate(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth < 4 {
		return s[:maxWidth]
	}
	return s[:maxWidth-3] + "..."
}

// Pluralize returns the singular or plural form based on count.
func Pluralize(count int, singular string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %ss", count, singular)
}

// RelativeTime returns a human-readable relative time string.
func RelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		m := int(diff.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case diff < 24*time.Hour:
		h := int(diff.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case diff < 30*24*time.Hour:
		d := int(diff.Hours() / 24)
		if d == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", d)
	case diff < 365*24*time.Hour:
		m := int(diff.Hours() / 24 / 30)
		if m == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", m)
	default:
		y := int(diff.Hours() / 24 / 365)
		if y == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", y)
	}
}

// PriorityName returns the human-readable name for a ClickUp priority level.
func PriorityName(priority int) string {
	switch priority {
	case 1:
		return "Urgent"
	case 2:
		return "High"
	case 3:
		return "Normal"
	case 4:
		return "Low"
	default:
		return "None"
	}
}

// IndentLines indents each line of text by the given prefix.
func IndentLines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
