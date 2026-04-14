package chat

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type deleteOptions struct {
	factory   *cmdutil.Factory
	messageID string
	confirm   bool
}

// NewCmdDelete returns the "chat delete" command.
func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "delete <message-id>",
		Short: "Delete a Chat message",
		Long: `Delete a message from a ClickUp Chat channel.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.`,
		Example: `  # Delete a message (with confirmation)
  clickup chat delete msg123

  # Delete without confirmation
  clickup chat delete msg123 --yes`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.messageID = args[0]
			return runDelete(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(opts *deleteOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	if !opts.confirm && ios.IsTerminal() {
		p := prompter.New(ios)
		ok, err := p.Confirm(fmt.Sprintf("Delete message %s?", opts.messageID), false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(ios.ErrOut, "Cancelled.")
			return nil
		}
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		return err
	}
	if cfg.Workspace == "" {
		return fmt.Errorf("no workspace configured; run 'clickup auth' first")
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	if err := apiv3.DeleteChatMessage(context.Background(), client, cfg.Workspace, opts.messageID); err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Message %s deleted\n", cs.Green("!"), cs.Bold(opts.messageID))
	return nil
}
