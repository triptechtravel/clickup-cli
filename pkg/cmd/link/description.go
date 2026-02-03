package link

import (
	"context"
	"fmt"
	"strings"

	"github.com/raksul/go-clickup/clickup"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

const (
	descLinksStart = "--- GitHub Links (clickup-cli) ---"
	descLinksEnd   = "--- /GitHub Links ---"
)

// linkEntry represents a single link line in the description section.
// Prefix is used for deduplication (e.g. "PR: owner/repo#42").
// Line is the full formatted line (e.g. "PR: owner/repo#42 - Title (url)").
type linkEntry struct {
	Prefix string
	Line   string
}

// upsertDescriptionLinks fetches the task, upserts the link entry into the
// description's GitHub Links block, and updates the task.
func upsertDescriptionLinks(f *cmdutil.Factory, taskID string, entry linkEntry) error {
	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	task, _, err := client.Clickup.Tasks.GetTask(ctx, taskID, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch task for description update: %w", err)
	}

	newDesc := updateLinksBlock(task.Description, entry)

	updateReq := &clickup.TaskUpdateRequest{
		Description: newDesc,
	}
	_, _, err = client.Clickup.Tasks.UpdateTask(ctx, taskID, nil, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update task description: %w", err)
	}

	return nil
}

// updateLinksBlock takes the full task description and a new link entry,
// then returns the updated description with the entry upserted into the
// GitHub Links block.
func updateLinksBlock(description string, entry linkEntry) string {
	entries, rest := parseLinksBlock(description)

	// Deduplicate: replace existing line with same prefix, or append.
	found := false
	for i, existing := range entries {
		if strings.HasPrefix(existing, entry.Prefix) {
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
// from a task description that may contain a GitHub Links block.
func parseLinksBlock(description string) (entries []string, rest string) {
	startIdx := strings.Index(description, descLinksStart)
	endIdx := strings.Index(description, descLinksEnd)

	if startIdx < 0 || endIdx < 0 || endIdx <= startIdx {
		// No existing block.
		return nil, description
	}

	// Extract the block content between markers.
	blockContent := description[startIdx+len(descLinksStart) : endIdx]

	// Parse individual entries from the block.
	for _, line := range strings.Split(blockContent, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			entries = append(entries, line)
		}
	}

	// Reconstruct the rest of the description (before + after block).
	before := strings.TrimSpace(description[:startIdx])
	after := strings.TrimSpace(description[endIdx+len(descLinksEnd):])

	parts := []string{}
	if before != "" {
		parts = append(parts, before)
	}
	if after != "" {
		parts = append(parts, after)
	}
	rest = strings.Join(parts, "\n\n")

	return entries, rest
}

// buildLinksSection formats the link entries into a complete GitHub Links block.
func buildLinksSection(entries []string) string {
	var sb strings.Builder
	sb.WriteString(descLinksStart)
	sb.WriteString("\n")
	for _, entry := range entries {
		sb.WriteString(entry)
		sb.WriteString("\n")
	}
	sb.WriteString(descLinksEnd)
	return sb.String()
}
