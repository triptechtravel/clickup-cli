package chat

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	clickupv3 "github.com/triptechtravel/clickup-cli/api/clickupv3"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type reactOptions struct {
	messageID string
	emoji     string
}

// NewCmdReact returns the "chat react" command.
func NewCmdReact(f *cmdutil.Factory) *cobra.Command {
	opts := &reactOptions{}

	cmd := &cobra.Command{
		Use:   "react <message-id> <emoji>",
		Short: "Add a reaction to a Chat message",
		Long:  "Add an emoji reaction to a message in a ClickUp Chat channel.",
		Example: `  # React with thumbs up
  clickup chat react msg123 thumbsup

  # React with a custom emoji
  clickup chat react msg123 rocket`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.messageID = args[0]
			opts.emoji = args[1]
			return runReact(f, opts)
		},
	}

	return cmd
}

func runReact(f *cmdutil.Factory, opts *reactOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	cfg, err := f.Config()
	if err != nil {
		return err
	}
	if cfg.Workspace == "" {
		return fmt.Errorf("no workspace configured; run 'clickup auth' first")
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	req := &clickupv3.CommentCreateChatReaction{
		Reaction: opts.emoji,
	}

	_, err = apiv3.CreateChatReaction(context.Background(), client, cfg.Workspace, opts.messageID, req)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Reaction :%s: added to message %s\n", cs.Green("!"), opts.emoji, cs.Bold(opts.messageID))

	return nil
}
