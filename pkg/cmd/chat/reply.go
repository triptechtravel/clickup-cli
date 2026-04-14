package chat

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	clickupv3 "github.com/triptechtravel/clickup-cli/api/clickupv3"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type replyOptions struct {
	messageID string
	message   string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdReply returns the "chat reply" command.
func NewCmdReply(f *cmdutil.Factory) *cobra.Command {
	opts := &replyOptions{}

	cmd := &cobra.Command{
		Use:   "reply <message-id> <text>",
		Short: "Reply to a Chat message",
		Long:  "Reply to a message in a ClickUp Chat channel.",
		Example: `  # Reply to a message
  clickup chat reply msg123 "Got it, thanks!"

  # Reply and get JSON response
  clickup chat reply msg123 "On it" --json`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.messageID = args[0]
			opts.message = args[1]
			return runReply(f, opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runReply(f *cmdutil.Factory, opts *replyOptions) error {
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

	req := &clickupv3.CommentCreateChatMessage{
		Type:    "message",
		Content: opts.message,
	}
	req.ApplyDefaults()

	resp, err := apiv3.CreateReplyMessage(context.Background(), client, cfg.Workspace, opts.messageID, req)
	if err != nil {
		return fmt.Errorf("failed to reply: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, resp)
	}

	fmt.Fprintf(ios.Out, "%s Reply sent to message %s\n", cs.Green("!"), cs.Bold(opts.messageID))

	return nil
}
