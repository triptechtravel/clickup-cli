package link

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// resolveTaskResult holds the resolved task ID along with the git context
// (which may be nil if the task was resolved interactively or via --task flag
// without a git repository).
type resolveTaskResult struct {
	TaskID string
	GitCtx *git.RepoContext
}

// resolveTask determines the ClickUp task ID using a priority chain:
//  1. Explicit --task flag value
//  2. Auto-detection from the current git branch
//  3. Interactive prompt (search + select) if running in a TTY
//
// If none of the above succeed, it returns an error with a helpful message.
func resolveTask(f *cmdutil.Factory, flagTaskID string) (*resolveTaskResult, error) {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	// 1. Explicit --task flag.
	if flagTaskID != "" {
		fmt.Fprintf(ios.ErrOut, "Using task %s\n", cs.Bold(flagTaskID))
		// Still try to get git context for repo info, but don't fail if unavailable.
		gitCtx, _ := f.GitContext()
		return &resolveTaskResult{TaskID: flagTaskID, GitCtx: gitCtx}, nil
	}

	// 2. Auto-detect from git branch.
	gitCtx, gitErr := f.GitContext()
	if gitErr == nil && gitCtx.TaskID != nil {
		fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n",
			cs.Bold(gitCtx.TaskID.ID), cs.Cyan(gitCtx.Branch))
		return &resolveTaskResult{TaskID: gitCtx.TaskID.ID, GitCtx: gitCtx}, nil
	}

	// 3. Interactive fallback if TTY.
	if ios.IsTerminal() {
		branchHint := ""
		if gitErr == nil && gitCtx != nil {
			branchHint = gitCtx.Branch
		}
		if branchHint != "" {
			fmt.Fprintf(ios.ErrOut, "No task ID found in branch %s\n", cs.Cyan(branchHint))
		} else {
			fmt.Fprintf(ios.ErrOut, "Could not auto-detect a ClickUp task ID\n")
		}

		taskID, err := promptForTask(f)
		if err != nil {
			return nil, err
		}
		return &resolveTaskResult{TaskID: taskID, GitCtx: gitCtx}, nil
	}

	// Non-interactive: return a helpful error.
	if gitErr != nil {
		return nil, fmt.Errorf("could not detect git context: %w\n\nTip: use --task to specify the task ID, or run 'clickup task recent' to find your tasks", gitErr)
	}
	return nil, fmt.Errorf("%s\n\nTip: use --task to specify the task ID, or run 'clickup task recent' to find your tasks",
		git.BranchNamingSuggestion(gitCtx.Branch))
}

// promptForTask interactively prompts the user to find a task via search,
// manual entry, or cancellation.
func promptForTask(f *cmdutil.Factory) (string, error) {
	ios := f.IOStreams
	p := prompter.New(ios)

	idx, err := p.Select("How would you like to specify the task?", []string{
		"Search for a task by name",
		"Show my recent tasks",
		"Enter a task ID manually",
		"Cancel",
	})
	if err != nil {
		return "", err
	}

	switch idx {
	case 0:
		return promptSearchTask(f, p)
	case 1:
		return promptRecentTasks(f, p)
	case 2:
		return promptManualTaskID(p)
	default:
		return "", fmt.Errorf("cancelled")
	}
}

// promptSearchTask asks for a search query, searches the ClickUp API, and lets
// the user pick a task from the results.
func promptSearchTask(f *cmdutil.Factory, p *prompter.Prompter) (string, error) {
	ios := f.IOStreams

	query, err := p.Input("Search query:", "")
	if err != nil {
		return "", err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("no search query provided")
	}

	fmt.Fprintf(ios.ErrOut, "Searching for %q...\n", query)

	tasks, err := searchTasks(f, query)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Fprintf(ios.ErrOut, "No tasks found matching %q\n", query)
		// Offer recent tasks, retry, or manual entry.
		retryIdx, err := p.Select("What would you like to do?", []string{
			"Show my recent tasks",
			"Try a different search",
			"Enter a task ID manually",
			"Cancel",
		})
		if err != nil {
			return "", err
		}
		switch retryIdx {
		case 0:
			return promptRecentTasks(f, p)
		case 1:
			return promptSearchTask(f, p)
		case 2:
			return promptManualTaskID(p)
		default:
			return "", fmt.Errorf("cancelled")
		}
	}

	// Build selection options from results.
	options := make([]string, len(tasks))
	for i, t := range tasks {
		id := t.ID
		if t.CustomID != "" {
			id = t.CustomID
		}
		options[i] = fmt.Sprintf("[%s] %s (%s)", id, t.Name, t.Status.Status)
	}
	options = append(options, "Enter task ID manually", "Cancel")

	selected, err := p.Select("Select a task:", options)
	if err != nil {
		return "", err
	}

	if selected == len(options)-1 { // Cancel
		return "", fmt.Errorf("cancelled")
	}
	if selected == len(options)-2 { // Manual entry
		return promptManualTaskID(p)
	}

	// Return the selected task ID.
	t := tasks[selected]
	id := t.ID
	if t.CustomID != "" {
		id = t.CustomID
	}

	cs := ios.ColorScheme()
	fmt.Fprintf(ios.ErrOut, "Selected task %s\n", cs.Bold(id))
	return id, nil
}

// promptManualTaskID asks the user to type in a task ID.
func promptManualTaskID(p *prompter.Prompter) (string, error) {
	taskID, err := p.Input("Task ID:", "")
	if err != nil {
		return "", err
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return "", fmt.Errorf("no task ID provided")
	}
	return taskID, nil
}

// resolveSearchTask is a minimal task representation for search results
// within the link package.
type resolveSearchTask struct {
	ID       string `json:"id"`
	CustomID string `json:"custom_id"`
	Name     string `json:"name"`
	Status   struct {
		Status string `json:"status"`
	} `json:"status"`
}

type resolveSearchResponse struct {
	Tasks []resolveSearchTask `json:"tasks"`
}

// searchTasks searches the ClickUp API for tasks matching the given query.
// It uses the paginated team endpoint (GET /api/v2/team/{teamID}/task).
func searchTasks(f *cmdutil.Factory, query string) ([]resolveSearchTask, error) {
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

	ios := f.IOStreams
	queryLower := strings.ToLower(query)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var allTasks []resolveSearchTask
	for page := 0; page < 5; page++ {
		if ctx.Err() != nil {
			fmt.Fprintf(ios.ErrOut, "Search timed out\n")
			break
		}

		apiURL := fmt.Sprintf(
			"https://api.clickup.com/api/v2/team/%s/task?include_closed=true&page=%d&order_by=updated&reverse=true",
			teamID, page,
		)

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.DoRequest(req)
		if err != nil {
			return nil, fmt.Errorf("API request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			break
		}

		var result resolveSearchResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		if len(result.Tasks) == 0 {
			break
		}

		for _, t := range result.Tasks {
			if strings.Contains(strings.ToLower(t.Name), queryLower) {
				allTasks = append(allTasks, t)
			}
		}

		fmt.Fprintf(ios.ErrOut, "  scanned page %d (%d tasks)...\n", page+1, len(result.Tasks))

		// Stop early if we have enough results for interactive selection.
		if len(allTasks) >= 20 {
			break
		}
	}

	return allTasks, nil
}

// promptRecentTasks fetches the user's recent tasks and lets them pick one.
func promptRecentTasks(f *cmdutil.Factory, p *prompter.Prompter) (string, error) {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	fmt.Fprintln(ios.ErrOut, "Fetching your recent tasks...")
	tasks, err := cmdutil.FetchRecentTasks(f, 15)
	if err != nil {
		return "", fmt.Errorf("failed to fetch recent tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Fprintln(ios.ErrOut, "No recent tasks found.")
		return promptManualTaskID(p)
	}

	// Show location context.
	locations := cmdutil.LocationSummary(tasks)
	if len(locations) > 0 {
		fmt.Fprintf(ios.ErrOut, "Your active locations: %s\n", strings.Join(locations, ", "))
	}

	// Build selection options.
	options := make([]string, len(tasks))
	for i, t := range tasks {
		options[i] = cmdutil.FormatRecentTaskOption(t)
	}
	options = append(options, "Enter task ID manually", "Cancel")

	selected, err := p.Select("Select a task:", options)
	if err != nil {
		return "", err
	}

	if selected == len(options)-1 { // Cancel
		return "", fmt.Errorf("cancelled")
	}
	if selected == len(options)-2 { // Manual entry
		return promptManualTaskID(p)
	}

	t := tasks[selected]
	fmt.Fprintf(ios.ErrOut, "Selected task %s\n", cs.Bold(t.ID))
	return t.ID, nil
}
