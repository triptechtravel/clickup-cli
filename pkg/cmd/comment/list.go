package comment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type listOptions struct {
	factory   *cmdutil.Factory
	taskID    string
	jsonFlags cmdutil.JSONFlags
}

type commentResponse struct {
	Comments []commentData `json:"comments"`
}

type commentData struct {
	ID          string `json:"id"`
	CommentText string `json:"comment_text"`
	User        struct {
		Username string `json:"username"`
	} `json:"user"`
	Date string `json:"date"` // unix timestamp in ms as string
}

// NewCmdList returns the "comment list" command.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "list [TASK]",
		Short: "List comments on a task",
		Long: `List comments on a ClickUp task.

If TASK is not provided, the task ID is auto-detected from the current git branch.`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 1 {
				opts.taskID = args[0]
			}
			return listRun(opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func listRun(opts *listOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID from git branch if not provided.
	taskID := opts.taskID
	if taskID == "" {
		gitCtx, err := opts.factory.GitContext()
		if err != nil {
			return fmt.Errorf("could not detect git context: %w\n\n%s", err,
				"Tip: provide the task ID as the first argument")
		}
		if gitCtx.TaskID == nil {
			return fmt.Errorf("%s", git.BranchNamingSuggestion(gitCtx.Branch))
		}
		taskID = gitCtx.TaskID.ID
		fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n", cs.Bold(taskID), cs.Cyan(gitCtx.Branch))
	}

	// Fetch comments from the API.
	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s/comment", taskID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result commentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, result.Comments)
	}

	if len(result.Comments) == 0 {
		fmt.Fprintf(ios.ErrOut, "No comments found on task %s\n", taskID)
		return nil
	}

	tp := tableprinter.New(ios)
	tp.SetTruncateColumn(2)

	for _, c := range result.Comments {
		tp.AddField(c.User.Username)
		tp.AddField(formatCommentDate(c.Date))
		tp.AddField(text.Truncate(c.CommentText, 80))
		tp.EndRow()
	}

	return tp.Render()
}

// formatCommentDate converts a unix timestamp in milliseconds (as string) to a relative time.
func formatCommentDate(dateStr string) string {
	ms, err := strconv.ParseInt(dateStr, 10, 64)
	if err != nil {
		return dateStr
	}
	t := time.UnixMilli(ms)
	return text.RelativeTime(t)
}
