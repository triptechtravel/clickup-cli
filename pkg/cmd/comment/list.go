package comment

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
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
	Date       string        `json:"date"` // unix timestamp in ms as string
	ReplyCount int           `json:"reply_count"`
	Replies    []commentData `json:"replies,omitempty"`
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
	isCustomID := false
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
		isCustomID = gitCtx.TaskID.IsCustomID
		fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n", cs.Bold(taskID), cs.Cyan(gitCtx.Branch))
	} else {
		parsed := git.ParseTaskID(taskID)
		taskID = parsed.ID
		isCustomID = parsed.IsCustomID
	}

	// Fetch comments from the API.
	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	cfg, cfgErr := opts.factory.Config()
	if cfgErr != nil {
		return cfgErr
	}

	listPath := fmt.Sprintf("task/%s/comment%s", taskID, cmdutil.CustomIDQueryParam(cfg, isCustomID))
	ctx := context.Background()
	var result commentResponse
	if err := apiv2.Do(ctx, client, "GET", listPath, nil, &result); err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	// Fetch threaded replies for comments that have them.
	for i, c := range result.Comments {
		if c.ReplyCount > 0 {
			replies, err := fetchCommentReplies(client, c.ID)
			if err != nil {
				continue
			}
			result.Comments[i].Replies = replies
		}
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

		for _, r := range c.Replies {
			tp.AddField("  ↳ " + r.User.Username)
			tp.AddField(formatCommentDate(r.Date))
			tp.AddField(text.Truncate(r.CommentText, 80))
			tp.EndRow()
		}
	}

	if err := tp.Render(); err != nil {
		return err
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup comment add %s \"@user text\" (supports @mentions)\n", cs.Gray("Reply:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup comment list %s --json\n", cs.Gray("JSON:"), taskID)

	return nil
}

type commentRepliesResponse struct {
	Comments []commentData `json:"comments"`
	Replies  []commentData `json:"replies"`
}

func fetchCommentReplies(client *api.Client, commentID string) ([]commentData, error) {
	ctx := context.Background()
	var result commentRepliesResponse
	if err := apiv2.Do(ctx, client, "GET", fmt.Sprintf("comment/%s/reply", commentID), nil, &result); err != nil {
		return nil, err
	}

	if len(result.Replies) > 0 {
		return result.Replies, nil
	}
	return result.Comments, nil
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
