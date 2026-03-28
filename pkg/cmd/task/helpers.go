package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/raksul/go-clickup/clickup"
	"github.com/triptechtravel/clickup-cli/internal/api"
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

// setTaskPoints sets sprint/story points on a task via the ClickUp API.
// The go-clickup library's TaskRequest does not support points, so this
// makes a raw HTTP PUT request.
func setTaskPoints(client *api.Client, taskID string, points float64) error {
	body := fmt.Sprintf(`{"points":%g}`, points)
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.clickup.com/api/v2/task/%s", taskID), strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to set points: status %d", resp.StatusCode)
	}
	return nil
}

// addTaskTag adds a single tag to a task via the ClickUp tag API.
// The go-clickup library sends tags in the request body which ClickUp ignores;
// tags must be added via POST /task/{id}/tag/{tag_name}.
func addTaskTag(client *api.Client, taskID, tag string) error {
	req, err := http.NewRequest("POST",
		fmt.Sprintf("https://api.clickup.com/api/v2/task/%s/tag/%s", taskID, tag),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to add tag %q: status %d", tag, resp.StatusCode)
	}
	return nil
}

// removeTaskTag removes a single tag from a task via the ClickUp tag API.
func removeTaskTag(client *api.Client, taskID, tag string) error {
	req, err := http.NewRequest("DELETE",
		fmt.Sprintf("https://api.clickup.com/api/v2/task/%s/tag/%s", taskID, tag),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to remove tag %q: status %d", tag, resp.StatusCode)
	}
	return nil
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

// addTaskToList adds a task to an additional list via the ClickUp API.
// Uses POST /api/v2/list/{list_id}/task/{task_id}.
func addTaskToList(client *api.Client, listID, taskID string) error {
	req, err := http.NewRequest("POST",
		fmt.Sprintf("https://api.clickup.com/api/v2/list/%s/task/%s", listID, taskID),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to add task to list: status %d", resp.StatusCode)
	}
	return nil
}

// removeTaskFromList removes a task from an additional list via the ClickUp API.
// Uses DELETE /api/v2/list/{list_id}/task/{task_id}.
func removeTaskFromList(client *api.Client, listID, taskID string) error {
	req, err := http.NewRequest("DELETE",
		fmt.Sprintf("https://api.clickup.com/api/v2/list/%s/task/%s", listID, taskID),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to remove task from list: status %d", resp.StatusCode)
	}
	return nil
}

// setMarkdownDescription sets the markdown_description field on a task
// via a raw HTTP PUT request. The go-clickup library's TaskUpdateRequest
// does not support this field.
func setMarkdownDescription(client *api.Client, taskID string, md string) error {
	payload, err := json.Marshal(map[string]string{"markdown_description": md})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.clickup.com/api/v2/task/%s", taskID), strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to set markdown description: status %d", resp.StatusCode)
	}
	return nil
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

	listID, err := cmdutil.ResolveCurrentSprintListID(context.Background(), client.Clickup, folderID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve current sprint: %w", err)
	}
	if listID == "" {
		return "", fmt.Errorf("no active sprint found in folder %s. Use --list-id to specify a list", folderID)
	}

	return listID, nil
}

// resolveListByName searches all lists (folderless + inside folders) in the
// configured space and returns the list ID matching the given name.
// Matching is case-insensitive.
func resolveListByName(f *cmdutil.Factory, name string) (string, error) {
	cfg, err := f.Config()
	if err != nil {
		return "", err
	}

	spaceID := cfg.Space
	if spaceID == "" {
		return "", fmt.Errorf("no space configured. Run 'clickup space select' first")
	}

	client, err := f.ApiClient()
	if err != nil {
		return "", err
	}

	nameLower := strings.ToLower(name)

	type apiList struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	// Check folderless lists first.
	if listID, err := findListInURL(client, fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/list", spaceID), nameLower); err != nil {
		return "", err
	} else if listID != "" {
		return listID, nil
	}

	// Check folders.
	type apiFolder struct {
		Lists []apiList `json:"lists"`
	}

	foldersURL := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/folder", spaceID)
	req, err := http.NewRequest(http.MethodGet, foldersURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.DoRequest(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch folders: %s", string(body))
	}

	var foldersResult struct {
		Folders []apiFolder `json:"folders"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&foldersResult); err != nil {
		return "", err
	}

	for _, folder := range foldersResult.Folders {
		for _, l := range folder.Lists {
			if strings.ToLower(l.Name) == nameLower {
				return l.ID, nil
			}
		}
	}

	return "", fmt.Errorf("list %q not found in space %s. Run 'clickup list ls' to see available lists", name, spaceID)
}

// findListInURL fetches lists from a ClickUp API URL and returns the ID of
// the list matching nameLower, or empty string if not found.
func findListInURL(client *api.Client, url, nameLower string) (string, error) {
	type apiList struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.DoRequest(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch lists: %s", string(body))
	}

	var result struct {
		Lists []apiList `json:"lists"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	for _, l := range result.Lists {
		if strings.ToLower(l.Name) == nameLower {
			return l.ID, nil
		}
	}
	return "", nil
}
