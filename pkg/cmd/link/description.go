package link

import (
	"context"
	"fmt"
	"strings"

	"github.com/raksul/go-clickup/clickup"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

const (
	// Markdown header used as the start marker for the links block.
	// Renders as bold + italic in ClickUp.
	descLinksHeader = "**GitHub** _(clickup-cli)_"
)

// markdownDescUpdate is a minimal struct for updating the task description
// via the markdown_description API field, which ClickUp renders as rich text.
type markdownDescUpdate struct {
	MarkdownDescription string `json:"markdown_description"`
}

// linkEntry represents a single link line in the description section.
// Prefix is used for deduplication (matched via Contains).
// Line is the full formatted line (without bullet prefix).
type linkEntry struct {
	Prefix string
	Line   string
}

// upsertDescriptionLinks fetches the task's markdown description, upserts
// the link entry, and writes back via markdown_description for rich rendering.
func upsertDescriptionLinks(f *cmdutil.Factory, taskID string, entry linkEntry) error {
	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Fetch task with include_markdown_description=true to get the markdown source.
	desc, err := fetchMarkdownDescription(ctx, client.Clickup, taskID)
	if err != nil {
		return fmt.Errorf("failed to fetch task for description update: %w", err)
	}

	newDesc := updateLinksBlock(desc, entry)

	// Write via markdown_description so ClickUp renders markdown as rich text.
	body := &markdownDescUpdate{MarkdownDescription: newDesc}
	req, err := client.Clickup.NewRequest("PUT", fmt.Sprintf("task/%s/", taskID), body)
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}

	_, err = client.Clickup.Do(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("failed to update task description: %w", err)
	}

	return nil
}

// fetchMarkdownDescription fetches the task with include_markdown_description=true
// and returns the markdown description (falling back to plain description).
func fetchMarkdownDescription(ctx context.Context, c *clickup.Client, taskID string) (string, error) {
	var task clickup.Task
	req, err := c.NewRequest("GET", fmt.Sprintf("task/%s/?include_markdown_description=true", taskID), nil)
	if err != nil {
		return "", err
	}
	_, err = c.Do(ctx, req, &task)
	if err != nil {
		return "", err
	}
	if task.MarkdownDescription != "" {
		return task.MarkdownDescription, nil
	}
	return task.Description, nil
}

// updateLinksBlock takes the full task description and a new link entry,
// then returns the updated description with the entry upserted into the
// GitHub Links block.
func updateLinksBlock(description string, entry linkEntry) string {
	entries, rest := parseLinksBlock(description)

	// Deduplicate: replace existing entry containing the same prefix, or append.
	found := false
	for i, existing := range entries {
		if strings.Contains(existing, entry.Prefix) {
			entries[i] = entry.Line
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, entry.Line)
	}

	section := buildLinksSection(entries)

	rest = strings.TrimSpace(rest)
	if rest == "" {
		return section
	}
	return section + "\n\n" + rest
}

// parseLinksBlock extracts the link entries and the remaining description
// from a task description that contains a GitHub Links block.
func parseLinksBlock(description string) (entries []string, rest string) {
	headerIdx := strings.Index(description, descLinksHeader)
	if headerIdx < 0 {
		return nil, description
	}

	before := description[:headerIdx]
	afterHeader := description[headerIdx+len(descLinksHeader):]

	// Collect entries: skip leading blank lines, then take consecutive
	// non-blank lines. Stop at first blank line after entries begin.
	lines := strings.Split(afterHeader, "\n")
	consumedLines := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			consumedLines = i + 1
			if len(entries) > 0 {
				break // Blank line after entries = end of block.
			}
			continue // Skip leading blanks.
		}
		// Strip bullet prefix: we write "- " but ClickUp's markdown export
		// uses "*   ". Handle both.
		if strings.HasPrefix(trimmed, "- ") {
			trimmed = trimmed[2:]
		} else if strings.HasPrefix(trimmed, "* ") {
			trimmed = strings.TrimLeft(trimmed[1:], " ")
		}
		entries = append(entries, trimmed)
		consumedLines = i + 1
	}

	remaining := strings.Join(lines[consumedLines:], "\n")
	beforeStr := strings.TrimSpace(before)
	remaining = strings.TrimSpace(remaining)

	var parts []string
	if beforeStr != "" {
		parts = append(parts, beforeStr)
	}
	if remaining != "" {
		parts = append(parts, remaining)
	}
	rest = strings.Join(parts, "\n\n")

	return entries, rest
}

// buildLinksSection formats the link entries into a markdown GitHub Links block.
func buildLinksSection(entries []string) string {
	var sb strings.Builder
	sb.WriteString(descLinksHeader)
	sb.WriteString("\n")
	for _, entry := range entries {
		sb.WriteString("- ")
		sb.WriteString(entry)
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
