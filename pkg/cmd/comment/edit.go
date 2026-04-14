package comment

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type editOptions struct {
	factory   *cmdutil.Factory
	commentID string
	body      string
	editor    bool
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
If BODY is not provided (or --editor is used), your editor opens for composing the new text.`,
		Args:              cobra.RangeArgs(1, 2),
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

	ctx := context.Background()
	payload := map[string]string{"comment_text": body}
	if err := apiv2.Do(ctx, client, "PUT", fmt.Sprintf("comment/%s", opts.commentID), payload, nil); err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Comment %s updated\n", cs.Green("!"), cs.Bold(opts.commentID))
	return nil
}
