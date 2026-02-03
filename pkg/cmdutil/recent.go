package cmdutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/raksul/go-clickup/clickup"
	"github.com/triptechtravel/clickup-cli/internal/api"
)

// RecentTask represents a recently updated task with location context.
type RecentTask struct {
	ID         string `json:"id"`
	CustomID   string `json:"custom_id,omitempty"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	ListName   string `json:"list_name"`
	FolderName string `json:"folder_name"`
}

// FetchRecentTasks fetches the current user's recently updated tasks.
// Uses 2 API calls: one for user identity, one for filtered tasks.
// Returns up to `limit` tasks ordered by most recently updated.
func FetchRecentTasks(f *Factory, limit int) ([]RecentTask, error) {
	client, err := f.ApiClient()
	if err != nil {
		return nil, err
	}

	cfg, err := f.Config()
	if err != nil {
		return nil, err
	}

	teamID := cfg.Workspace
	if teamID == "" {
		return nil, fmt.Errorf("workspace ID required. Set with 'clickup auth login'")
	}

	// Get current user ID (1 API call).
	userID, err := getCurrentUserID(client)
	if err != nil {
		// Fall back to unfiltered recent tasks if we can't get user.
		return fetchRecentTeamTasks(client, teamID, nil, limit)
	}

	assignees := []string{strconv.Itoa(userID)}
	return fetchRecentTeamTasks(client, teamID, assignees, limit)
}

// FetchRecentTeamTasks fetches recently updated tasks for the whole team (no assignee filter).
// Uses 1 API call.
func FetchRecentTeamTasks(f *Factory, limit int) ([]RecentTask, error) {
	client, err := f.ApiClient()
	if err != nil {
		return nil, err
	}

	cfg, err := f.Config()
	if err != nil {
		return nil, err
	}

	teamID := cfg.Workspace
	if teamID == "" {
		return nil, fmt.Errorf("workspace ID required. Set with 'clickup auth login'")
	}

	return fetchRecentTeamTasks(client, teamID, nil, limit)
}

func fetchRecentTeamTasks(client *api.Client, teamID string, assignees []string, limit int) ([]RecentTask, error) {
	ctx := context.Background()

	taskOpts := &clickup.GetTasksOptions{
		OrderBy:       "updated",
		Reverse:       true,
		IncludeClosed: false,
		Subtasks:      true,
	}
	if len(assignees) > 0 {
		taskOpts.Assignees = assignees
	}

	tasks, _, err := client.Clickup.Tasks.GetFilteredTeamTasks(ctx, teamID, taskOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}

	if limit > 0 && len(tasks) > limit {
		tasks = tasks[:limit]
	}

	results := make([]RecentTask, len(tasks))
	for i, t := range tasks {
		id := t.ID
		if t.CustomID != "" {
			id = t.CustomID
		}
		results[i] = RecentTask{
			ID:         id,
			CustomID:   t.CustomID,
			Name:       t.Name,
			Status:     t.Status.Status,
			ListName:   t.List.Name,
			FolderName: t.Folder.Name,
		}
	}

	return results, nil
}

// LocationSummary returns a deduplicated summary of locations from recent tasks,
// e.g. "Folder > List" pairs. Useful for suggesting where to search.
func LocationSummary(tasks []RecentTask) []string {
	seen := make(map[string]bool)
	var locations []string
	for _, t := range tasks {
		var loc string
		if t.FolderName != "" && t.ListName != "" {
			loc = t.FolderName + " > " + t.ListName
		} else if t.FolderName != "" {
			loc = t.FolderName
		} else if t.ListName != "" {
			loc = t.ListName
		}
		if loc != "" && !seen[loc] {
			seen[loc] = true
			locations = append(locations, loc)
		}
	}
	return locations
}

// FormatRecentTaskOption formats a RecentTask for display in a selection prompt.
func FormatRecentTaskOption(t RecentTask) string {
	parts := []string{fmt.Sprintf("[%s] %s (%s)", t.ID, t.Name, t.Status)}
	if t.FolderName != "" || t.ListName != "" {
		loc := ""
		if t.FolderName != "" {
			loc = t.FolderName
		}
		if t.ListName != "" {
			if loc != "" {
				loc += " > "
			}
			loc += t.ListName
		}
		parts = append(parts, fmt.Sprintf("in %s", loc))
	}
	return strings.Join(parts, " ")
}

type userResp struct {
	User struct {
		ID int `json:"id"`
	} `json:"user"`
}

func getCurrentUserID(client *api.Client) (int, error) {
	req, err := http.NewRequest("GET", "https://api.clickup.com/api/v2/user", nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.DoRequest(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result userResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.User.ID, nil
}
