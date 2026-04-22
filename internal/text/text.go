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

// FormatUnixMillis converts a Unix millisecond timestamp string (as returned
// by the ClickUp v3 API) into a human-readable relative time. Returns the
// original string unchanged if it cannot be parsed.
func FormatUnixMillis(ms string) string {
	if ms == "" {
		return ""
	}
	var millis int64
	if _, err := fmt.Sscan(ms, &millis); err != nil {
		return ms
	}
	t := time.UnixMilli(millis)
	return RelativeTime(t)
}

// FormatUnixMillisFloat converts a Unix millisecond timestamp stored as float32
// (as used in auto-generated clickupv3 types) into a human-readable relative
// time. Returns an empty string for zero values.
func FormatUnixMillisFloat(ms float32) string {
	if ms == 0 {
		return ""
	}
	t := time.UnixMilli(int64(ms))
	return RelativeTime(t)
}

// FormatBytes returns a human-readable byte size string (e.g. "1.2 MB").
func FormatBytes(bytes int) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// SplitAndTrim splits s on commas, trims whitespace from each token, and
// drops empty tokens.
func SplitAndTrim(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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
