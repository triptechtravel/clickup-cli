package task

import (
	"context"
	"fmt"
	"time"

	"github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// parseDate converts a YYYY-MM-DD string to a *clickup.Date.
func parseDate(s string) (*clickup.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q (use YYYY-MM-DD format): %w", s, err)
	}
	return clickup.NewDate(t), nil
}

// parseDuration converts a human-readable duration string (e.g. "2h", "30m", "1h30m")
// to milliseconds.
func parseDuration(s string) (int, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q (use format like 2h, 30m, 1h30m): %w", s, err)
	}
	return int(d.Milliseconds()), nil
}

// setTaskPoints sets sprint/story points on a task using the auto-generated
// UpdateTask wrapper (go-clickup's TaskUpdateRequest doesn't support points).
func setTaskPoints(client *api.Client, taskID string, points float64) error {
	p := float32(points)
	_, err := apiv2.UpdateTask(context.Background(), client, taskID, &clickupv2.UpdateTaskJSONRequest{
		Points: &p,
	})
	return err
}

// addTaskTag adds a single tag to a task via POST /task/{id}/tag/{name}.
func addTaskTag(client *api.Client, taskID, tag string) error {
	_, err := apiv2.AddTagToTask(context.Background(), client, taskID, tag)
	return err
}

// removeTaskTag removes a single tag from a task.
func removeTaskTag(client *api.Client, taskID, tag string) error {
	_, err := apiv2.RemoveTagFromTask(context.Background(), client, taskID, tag)
	return err
}

// addTaskTags adds multiple tags to a task. Used after task creation.
func addTaskTags(client *api.Client, taskID string, tags []string) error {
	for _, tag := range tags {
		if err := addTaskTag(client, taskID, tag); err != nil {
			return err
		}
	}
	return nil
}

// diffTags computes which tags to add and which to remove to go from
// currentTags to desiredTags.
func diffTags(currentTags, desiredTags []string) (toAdd, toRemove []string) {
	current := make(map[string]bool, len(currentTags))
	for _, t := range currentTags {
		current[t] = true
	}
	desired := make(map[string]bool, len(desiredTags))
	for _, t := range desiredTags {
		desired[t] = true
	}

	for _, t := range currentTags {
		if !desired[t] {
			toRemove = append(toRemove, t)
		}
	}
	for _, t := range desiredTags {
		if !current[t] {
			toAdd = append(toAdd, t)
		}
	}
	return
}

// setTaskTags replaces all tags on a task with the desired set.
// It diffs current vs desired to minimise API calls.
func setTaskTags(client *api.Client, taskID string, currentTags, desiredTags []string) error {
	toAdd, toRemove := diffTags(currentTags, desiredTags)

	for _, t := range toRemove {
		if err := removeTaskTag(client, taskID, t); err != nil {
			return err
		}
	}
	for _, t := range toAdd {
		if err := addTaskTag(client, taskID, t); err != nil {
			return err
		}
	}
	return nil
}

// addTaskToList adds a task to an additional list.
func addTaskToList(client *api.Client, listID, taskID string) error {
	_, err := apiv2.AddTaskToList(context.Background(), client, listID, taskID)
	return err
}

// removeTaskFromList removes a task from an additional list.
func removeTaskFromList(client *api.Client, listID, taskID string) error {
	_, err := apiv2.RemoveTaskFromList(context.Background(), client, listID, taskID)
	return err
}

// setMarkdownDescription sets the markdown description on a task using the
// auto-generated UpdateTask wrapper.
func setMarkdownDescription(client *api.Client, taskID string, md string) error {
	_, err := apiv2.UpdateTask(context.Background(), client, taskID, &clickupv2.UpdateTaskJSONRequest{
		MarkdownContent: &md,
	})
	return err
}

// resolveCurrentSprintList resolves the current sprint's list ID from the
// configured sprint folder. Returns an error if no sprint folder is configured
// or no active sprint is found.
func resolveCurrentSprintList(f *cmdutil.Factory) (string, error) {
	cfg, err := f.Config()
	if err != nil {
		return "", err
	}

	folderID := cfg.SprintFolder
	if folderID == "" {
		return "", fmt.Errorf("no sprint folder configured. Run 'clickup sprint current' first or use --list-id")
	}

	client, err := f.ApiClient()
	if err != nil {
		return "", err
	}

	listID, err := cmdutil.ResolveCurrentSprintListID(context.Background(), client, folderID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve current sprint: %w", err)
	}
	if listID == "" {
		return "", fmt.Errorf("no active sprint found in folder %s. Use --list-id to specify a list", folderID)
	}

	return listID, nil
}
