package chat

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type messagesOptions struct {
	channelID string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdMessages returns the "chat messages" command.
func NewCmdMessages(f *cmdutil.Factory) *cobra.Command {
	opts := &messagesOptions{}

	cmd := &cobra.Command{
		Use:   "messages <channel-id>",
		Short: "List messages in a Chat channel",
		Long:  "List messages in a ClickUp Chat channel.",
		Example: `  # List messages in a channel
  clickup chat messages abc123

  # List messages as JSON
  clickup chat messages abc123 --json`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channelID = args[0]
			return runMessages(f, opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runMessages(f *cmdutil.Factory, opts *messagesOptions) error {
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

	result, err := apiv3.ListChatMessages(context.Background(), client, cfg.Workspace, opts.channelID)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, result)
	}

	if len(result.Data) == 0 {
		fmt.Fprintln(ios.Out, cs.Gray("No messages found."))
		return nil
	}

	tp := tableprinter.New(ios)
	for _, msg := range result.Data {
		tp.AddField(msg.ID)
		tp.AddField(msg.UserID)
		tp.AddField(text.Truncate(msg.Content, 60))
		tp.AddField(text.FormatUnixMillisFloat(msg.Date))
		tp.EndRow()
	}
	return tp.Render()
}
