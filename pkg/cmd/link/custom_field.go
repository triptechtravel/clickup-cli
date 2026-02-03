package link

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/raksul/go-clickup/clickup"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// upsertLink routes the link entry to either the custom field approach
// (if link_field is configured) or the description approach (default).
func upsertLink(f *cmdutil.Factory, taskID string, entry linkEntry) error {
	cfg, err := f.Config()
	if err != nil {
		return err
	}

	// Check for directory-level override first, then workspace-level.
	linkField := ""
	if cwd, err := os.Getwd(); err == nil {
		linkField = cfg.LinkFieldForDir(cwd)
	}
	if linkField == "" {
		linkField = cfg.LinkField
	}

	if linkField != "" {
		return upsertCustomFieldLink(f, taskID, linkField, entry)
	}

	return upsertDescriptionLinks(f, taskID, entry)
}

// upsertCustomFieldLink updates a custom field on the task with the link entry.
// For URL-type fields, it sets the value to the URL extracted from the entry line.
// For text-type fields, it upserts the entry into the field's text value.
func upsertCustomFieldLink(f *cmdutil.Factory, taskID string, fieldName string, entry linkEntry) error {
	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	task, _, err := client.Clickup.Tasks.GetTask(ctx, taskID, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch task: %w", err)
	}

	// Find the custom field by name (case-insensitive).
	var fieldID string
	var fieldType string
	var currentValue interface{}
	for _, cf := range task.CustomFields {
		if strings.EqualFold(cf.Name, fieldName) {
			fieldID = cf.ID
			fieldType = cf.Type
			currentValue = cf.Value
			break
		}
	}

	if fieldID == "" {
		return fmt.Errorf("custom field %q not found on task %s\n\nAvailable custom fields: %s",
			fieldName, taskID, listCustomFieldNames(task.CustomFields))
	}

	var value map[string]interface{}

	switch fieldType {
	case "url":
		// URL fields: extract URL from the entry line (last parenthesized URL).
		url := extractURL(entry.Line)
		if url == "" {
			// No URL in entry (e.g., branch links). Fall back to description approach.
			return upsertDescriptionLinks(f, taskID, entry)
		}
		value = map[string]interface{}{"value": url}

	case "short_text", "text":
		// Text fields: upsert into multiline text value.
		existing := ""
		if currentValue != nil {
			if s, ok := currentValue.(string); ok {
				existing = s
			}
		}
		newText := upsertTextFieldEntry(existing, entry)
		value = map[string]interface{}{"value": newText}

	default:
		return fmt.Errorf("custom field %q has type %q which is not supported for link storage (use url, text, or short_text)", fieldName, fieldType)
	}

	_, err = client.Clickup.CustomFields.SetCustomFieldValue(ctx, task.ID, fieldID, value, nil)
	if err != nil {
		return fmt.Errorf("failed to set custom field %q: %w", fieldName, err)
	}

	return nil
}

// upsertTextFieldEntry upserts a link entry into a text field value,
// using the same prefix-based deduplication as the description approach.
func upsertTextFieldEntry(existing string, entry linkEntry) string {
	lines := strings.Split(existing, "\n")

	found := false
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, entry.Prefix) {
			result = append(result, entry.Line)
			found = true
		} else {
			result = append(result, trimmed)
		}
	}
	if !found {
		result = append(result, entry.Line)
	}

	return strings.Join(result, "\n")
}

// extractURL extracts a URL from a line, looking for the last (url) pattern.
func extractURL(line string) string {
	// Look for (https://...) pattern at the end.
	lastOpen := strings.LastIndex(line, "(https://")
	if lastOpen < 0 {
		lastOpen = strings.LastIndex(line, "(http://")
	}
	if lastOpen < 0 {
		return ""
	}

	lastClose := strings.LastIndex(line, ")")
	if lastClose <= lastOpen {
		return ""
	}

	return line[lastOpen+1 : lastClose]
}

// listCustomFieldNames returns a comma-separated list of custom field names.
func listCustomFieldNames(fields []clickup.CustomField) string {
	if len(fields) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.Name)
	}
	return strings.Join(names, ", ")
}
