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

type replyOptions struct {
	factory   *cmdutil.Factory
	commentID string
	body      string
	editor    bool
}

// NewCmdReply returns the "comment reply" command.
func NewCmdReply(f *cmdutil.Factory) *cobra.Command {
	opts := &replyOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "reply <comment-id> [BODY]",
		Short: "Reply to a comment thread",
		Long: `Reply to an existing comment on a ClickUp task, creating a threaded reply.

Use 'clickup comment list <task> --json' to find comment IDs.

The body is parsed as markdown — headers (##), bold (**x**), italic (*x*),
inline code, fenced code blocks, ordered/bullet lists, blockquotes, and links
all render as rich formatting.

Use @username in the body to @mention workspace members. Mentions resolve
case-insensitively against full username, first-name token, or email
local-part when unambiguous.`,
		Example: `  # Reply to a specific comment
  clickup comment reply 90160175975219 "Yes, that's confirmed"

  # Reply with @mentions
  clickup comment reply 90160175975219 "@Michelle confirmed, BookEasy only"

  # Open editor to compose the reply
  clickup comment reply 90160175975219 --editor`,
		Args:              cobra.RangeArgs(1, 2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.commentID = args[0]
			if len(args) >= 2 {
				opts.body = args[1]
			}
			return replyRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.editor, "editor", "e", false, "Open editor to compose reply body")

	return cmd
}

func replyRun(opts *replyOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve reply body.
	body := opts.body
	if body == "" || opts.editor {
		p := prompter.New(ios)
		var err error
		body, err = p.Editor("Reply body", body, "*.md")
		if err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}
	}

	if body == "" {
		return fmt.Errorf("reply body cannot be empty")
	}

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

	req := &clickupv2.CreateThreadedCommentJSONRequest{}
	if useBlocks {
		req.Comment = toReplyBlocks(blocks)
	} else {
		req.CommentText = &body
	}

	ctx := context.Background()
	if _, err := apiv2.CreateThreadedComment(ctx, client, opts.commentID, req); err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Reply added to comment %s\n", cs.Green("!"), cs.Bold(opts.commentID))

	return nil
}
