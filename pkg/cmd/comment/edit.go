package comment

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type editOptions struct {
	factory   *cmdutil.Factory
	commentID string
	body      string
	editor    bool
	jsonFlags cmdutil.JSONFlags
}

// editOutput mirrors addOutput for symmetry. UpdateComment returns no body
// so we just echo the comment ID and the resolved mentions.
type editOutput struct {
	ID               string   `json:"id"`
	ResolvedMentions []string `json:"resolved_mentions,omitempty"`
}

// NewCmdEdit returns the "comment edit" command.
func NewCmdEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &editOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "edit <COMMENT_ID> [BODY]",
		Short: "Edit a comment",
		Long: `Edit an existing comment on a ClickUp task.

COMMENT_ID is required as the first argument.
If BODY is not provided (or --editor is used), your editor opens for composing the new text.

The body is parsed as markdown — headers (##), bold (**x**), italic (*x*),
inline code, fenced code blocks, ordered/bullet lists, blockquotes, and links
all render as rich formatting.

Use @username in the body to @mention workspace members. Mentions resolve
case-insensitively against full username, first-name token, or email
local-part when unambiguous.`,
		Example: `  # Edit a comment
  clickup comment edit 90160175975219 "Updated the description"

  # Edit using your editor
  clickup comment edit 90160175975219 --editor

  # Re-add a mention via shortcut
  clickup comment edit 90160175975219 "Hey @alice — pushed the fix"`,
		Args: cobra.RangeArgs(1, 2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.commentID = args[0]
			if len(args) >= 2 {
				opts.body = args[1]
			}
			return editRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.editor, "editor", "e", false, "Open editor to compose comment body")
	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func editRun(opts *editOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

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

	// Build and send the API request.
	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	members, mErr := resolveMentionMembers(opts.factory, client, body)
	if mErr != nil {
		fmt.Fprintf(ios.ErrOut, "%s could not resolve @mentions: %v\n", cs.Yellow("warning:"), mErr)
	}
	blocks, resolved, useBlocks := buildBlocks(body, members)
	for _, name := range resolved {
		fmt.Fprintf(ios.ErrOut, "Mentioning %s\n", cs.Bold("@"+name))
	}

	req := &clickupv2.UpdateCommentJSONRequest{}
	if useBlocks {
		req.Comment = toUpdateBlocks(blocks)
	} else {
		req.CommentText = &body
	}

	ctx := context.Background()
	if _, err := apiv2.UpdateComment(ctx, client, opts.commentID, req); err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, editOutput{
			ID:               opts.commentID,
			ResolvedMentions: resolved,
		})
	}

	fmt.Fprintf(ios.Out, "%s Comment %s updated\n", cs.Green("!"), cs.Bold(opts.commentID))
	return nil
}
