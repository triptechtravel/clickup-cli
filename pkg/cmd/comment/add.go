package comment

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type addOptions struct {
	factory   *cmdutil.Factory
	taskID    string
	body      string
	editor    bool
	jsonFlags cmdutil.JSONFlags
}

// addOutput is the JSON-mode response — exposes the created comment's ID
// (so scripts can chain into edit/reply/delete without a follow-up list)
// plus the resolved mentions for confirmation.
type addOutput struct {
	ID               string   `json:"id"`
	HistID           string   `json:"hist_id"`
	Date             int      `json:"date"`
	ResolvedMentions []string `json:"resolved_mentions,omitempty"`
}

// NewCmdAdd returns the "comment add" command.
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	opts := &addOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "add [TASK] [BODY]",
		Short: "Add a comment to a task",
		Long: `Add a comment to a ClickUp task.

If TASK is not provided, the task ID is auto-detected from the current git branch.
If BODY is not provided (or --editor is used), your editor opens for composing the comment.

The body is parsed as markdown. Headers (##), bold (**x**), italic (*x*),
inline code, fenced code blocks, ordered/bullet lists, blockquotes, and links
are all rendered as rich formatting in ClickUp.

Use @username in the body to @mention workspace members. Mentions resolve
against your workspace member list (see 'clickup member list') with
case-insensitive matching, and additionally accept the first-name token or
email local-part when unambiguous within the workspace.`,
		Example: `  # Add a comment to the task detected from the current branch
  clickup comment add "" "Fixed the login bug"

  # Add a comment to a specific task
  clickup comment add abc123 "Deployed to staging"

  # Mention a teammate (triggers a real ClickUp notification)
  clickup comment add abc123 "Hey @alice can you review this?"

  # Mention multiple people
  clickup comment add abc123 "@alice @bob this is ready for QA"

  # Open your editor to compose the comment
  clickup comment add --editor`,
		Args:              cobra.MaximumNArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 1 {
				opts.taskID = args[0]
			}
			if len(args) >= 2 {
				opts.body = args[1]
			}
			return addRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.editor, "editor", "e", false, "Open editor to compose comment body")
	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func addRun(opts *addOptions) error {
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

	// Resolve comment body.
	body := opts.body
	if body == "" || opts.editor {
		p := prompter.New(ios)
		var err error
		body, err = p.Editor("Comment body", body, "*.md")
		if err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}
	}

	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	cfg, cfgErr := opts.factory.Config()
	if cfgErr != nil {
		return cfgErr
	}

	members, mErr := resolveMentionMembers(opts.factory, client, body)
	if mErr != nil {
		fmt.Fprintf(ios.ErrOut, "%s could not resolve @mentions: %v\n", cs.Yellow("warning:"), mErr)
	}
	blocks, resolved, useBlocks := buildBlocks(body, members)
	for _, name := range resolved {
		fmt.Fprintf(ios.ErrOut, "Mentioning %s\n", cs.Bold("@"+name))
	}

	req := &clickupv2.CreateTaskCommentJSONRequest{NotifyAll: true}
	if useBlocks {
		req.Comment = toCreateBlocks(blocks)
	} else {
		req.CommentText = &body
	}

	var params []apiv2.CreateTaskCommentParams
	if isCustomID {
		teamID, _ := strconv.ParseFloat(cfg.Workspace, 64)
		params = append(params, apiv2.CreateTaskCommentParams{
			CustomTaskIds: true,
			TeamId:        teamID,
		})
	}

	ctx := context.Background()
	resp, err := apiv2.CreateTaskComment(ctx, client, taskID, req, params...)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, addOutput{
			ID:               strconv.Itoa(resp.ID),
			HistID:           resp.HistID,
			Date:             resp.Date,
			ResolvedMentions: resolved,
		})
	}

	fmt.Fprintf(ios.Out, "%s Comment added to task %s\n", cs.Green("!"), cs.Bold(taskID))

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup comment list %s\n", cs.Gray("List:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task activity %s\n", cs.Gray("Activity:"), taskID)

	return nil
}
