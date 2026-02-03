package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type activityOptions struct {
	taskID    string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdActivity returns a command to view a task's details and comment history.
func NewCmdActivity(f *cmdutil.Factory) *cobra.Command {
	opts := &activityOptions{}

	cmd := &cobra.Command{
		Use:   "activity [<task-id>]",
		Short: "View a task's details and comment history",
		Long: `Display a task's details and all its comments in chronological order.

This gives a full picture of a task's change history by combining the task
summary (name, status, priority, assignees, dates) with every comment
posted on the task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. Branch names containing CU-<hex> or
PREFIX-<number> patterns are recognized.`,
		Example: `  # View activity for a specific task
  clickup task activity 86a3xrwkp

  # Auto-detect task from git branch
  clickup task activity

  # Output as JSON
  clickup task activity 86a3xrwkp --json

  # Filter with jq
  clickup task activity 86a3xrwkp --jq '.comments[] | .user'`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.taskID = args[0]
			}
			return runActivity(f, opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

// commentUser represents the user who posted a comment.
type commentUser struct {
	Username string `json:"username"`
}

// comment represents a single ClickUp task comment.
type comment struct {
	ID          string      `json:"id"`
	CommentText string      `json:"comment_text"`
	User        commentUser `json:"user"`
	Date        string      `json:"date"`
}

// commentsResponse represents the API response from the comments endpoint.
type commentsResponse struct {
	Comments []comment `json:"comments"`
}

// activityOutput is the structured output for JSON mode.
type activityOutput struct {
	Task     *clickup.Task `json:"task"`
	Comments []comment     `json:"comments"`
}

func runActivity(f *cmdutil.Factory, opts *activityOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	taskID := opts.taskID
	isCustomID := false

	// Auto-detect task ID from git branch if not provided.
	if taskID == "" {
		gitCtx, err := f.GitContext()
		if err != nil {
			return fmt.Errorf("could not detect task ID: %w\n\n%s", err, git.BranchNamingSuggestion(""))
		}
		if gitCtx.TaskID == nil {
			fmt.Fprintln(ios.ErrOut, cs.Yellow(git.BranchNamingSuggestion(gitCtx.Branch)))
			return &cmdutil.SilentError{Err: fmt.Errorf("no task ID found in branch")}
		}
		taskID = gitCtx.TaskID.ID
		isCustomID = gitCtx.TaskID.IsCustomID
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	var getOpts *clickup.GetTaskOptions
	if isCustomID {
		getOpts = &clickup.GetTaskOptions{
			CustomTaskIDs: true,
		}
	}

	ctx := context.Background()

	// Fetch task details.
	task, _, err := client.Clickup.Tasks.GetTask(ctx, taskID, getOpts)
	if err != nil {
		return fmt.Errorf("failed to fetch task %s: %w", taskID, err)
	}

	// Fetch comments via raw HTTP request.
	comments, err := fetchComments(client, taskID, isCustomID)
	if err != nil {
		return fmt.Errorf("failed to fetch comments for task %s: %w", taskID, err)
	}

	// Sort comments chronologically (oldest first).
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].Date < comments[j].Date
	})

	if opts.jsonFlags.WantsJSON() {
		output := activityOutput{
			Task:     task,
			Comments: comments,
		}
		return opts.jsonFlags.OutputJSON(ios.Out, output)
	}

	return printActivity(f, task, comments)
}

func fetchComments(client *api.Client, taskID string, isCustomID bool) ([]comment, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s/comment", taskID)
	if isCustomID {
		url += "?custom_task_ids=true"
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result commentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Comments, nil
}

// parseUnixMillis parses a unix timestamp in milliseconds (as a string) to a time.Time.
func parseUnixMillis(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, msInt*int64(time.Millisecond)), nil
}

func printActivity(f *cmdutil.Factory, task *clickup.Task, comments []comment) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	out := ios.Out

	// === Task Summary Header ===
	id := task.ID
	if task.CustomID != "" {
		id = task.CustomID
	}
	fmt.Fprintf(out, "%s %s\n", cs.Bold(task.Name), cs.Gray("#"+id))

	// Status
	statusText := task.Status.Status
	statusColorFn := cs.StatusColor(strings.ToLower(statusText))
	fmt.Fprintf(out, "%s %s\n", cs.Bold("Status:"), statusColorFn(statusText))

	// Priority
	if task.Priority.Priority != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Priority:"), task.Priority.Priority)
	}

	// Assignees
	if len(task.Assignees) > 0 {
		names := make([]string, 0, len(task.Assignees))
		for _, a := range task.Assignees {
			names = append(names, a.Username)
		}
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Assignees:"), strings.Join(names, ", "))
	}

	// Tags
	if len(task.Tags) > 0 {
		tagNames := make([]string, 0, len(task.Tags))
		for _, t := range task.Tags {
			tagNames = append(tagNames, t.Name)
		}
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Tags:"), strings.Join(tagNames, ", "))
	}

	// Points
	if pts := task.Points.Value.String(); pts != "" && pts != "0" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Points:"), pts)
	}

	// Time Estimate & Time Spent
	if s := formatMillisDuration(task.TimeEstimate); s != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Time Estimate:"), s)
	}
	if s := formatMillisDuration(task.TimeSpent); s != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Time Spent:"), s)
	}

	// Dates
	if task.DateCreated != "" {
		if t, err := parseUnixMillis(task.DateCreated); err == nil {
			fmt.Fprintf(out, "%s %s (%s)\n", cs.Bold("Created:"), t.Format("2006-01-02 15:04"), text.RelativeTime(t))
		}
	}
	if task.DateUpdated != "" {
		if t, err := parseUnixMillis(task.DateUpdated); err == nil {
			fmt.Fprintf(out, "%s %s (%s)\n", cs.Bold("Updated:"), t.Format("2006-01-02 15:04"), text.RelativeTime(t))
		}
	}
	if task.StartDate != "" {
		if t, err := parseUnixMillis(task.StartDate); err == nil {
			fmt.Fprintf(out, "%s %s (%s)\n", cs.Bold("Start Date:"), t.Format("2006-01-02 15:04"), text.RelativeTime(t))
		}
	}
	if task.DueDate != nil {
		if dt := task.DueDate.Time(); dt != nil {
			fmt.Fprintf(out, "%s %s (%s)\n", cs.Bold("Due:"), dt.Format("2006-01-02 15:04"), text.RelativeTime(*dt))
		}
	}

	// URL
	if task.URL != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("URL:"), cs.Cyan(task.URL))
	}

	// Description
	if task.Description != "" {
		fmt.Fprintf(out, "\n%s\n", cs.Bold("Description:"))
		fmt.Fprintf(out, "%s\n", text.IndentLines(task.Description, "  "))
	}

	// === Comments Section ===
	fmt.Fprintf(out, "\n%s\n", cs.Bold(fmt.Sprintf("Comments (%s):", text.Pluralize(len(comments), "comment"))))

	if len(comments) == 0 {
		fmt.Fprintf(out, "  %s\n", cs.Gray("No comments on this task."))
		return nil
	}

	for i, c := range comments {
		username := c.User.Username
		if username == "" {
			username = "Unknown"
		}

		var timeStr string
		if t, err := parseUnixMillis(c.Date); err == nil {
			timeStr = fmt.Sprintf("%s (%s)", t.Format("2006-01-02 15:04"), text.RelativeTime(t))
		} else {
			timeStr = c.Date
		}

		fmt.Fprintf(out, "\n  %s %s\n", cs.Bold(username), cs.Gray(timeStr))

		if c.CommentText != "" {
			fmt.Fprintf(out, "%s\n", text.IndentLines(c.CommentText, "    "))
		}

		// Print a separator between comments (but not after the last one).
		if i < len(comments)-1 {
			fmt.Fprintf(out, "  %s\n", cs.Gray("---"))
		}
	}

	return nil
}
