package webhook

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdWebhookDelete returns the webhook delete command.
func NewCmdWebhookDelete(f *cmdutil.Factory) *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <webhook-id>",
		Short: "Delete a webhook",
		Long: `Delete a ClickUp webhook permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.`,
		Example: `  # Delete a webhook (with confirmation)
  clickup webhook delete 4b67ac88

  # Delete without confirmation
  clickup webhook delete 4b67ac88 --yes`,
		Args:    cobra.ExactArgs(1),
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			webhookID := args[0]
			ios := f.IOStreams
			cs := ios.ColorScheme()

			if !confirm && ios.IsTerminal() {
				p := prompter.New(ios)
				ok, err := p.Confirm(fmt.Sprintf("Delete webhook %s?", webhookID), false)
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(ios.ErrOut, "Cancelled.")
					return nil
				}
			}

			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			_, err = apiv2.DeleteWebhook(context.Background(), client, webhookID)
			if err != nil {
				return fmt.Errorf("failed to delete webhook: %w", err)
			}

			fmt.Fprintf(ios.Out, "%s Webhook deleted (%s)\n", cs.Green("!"), webhookID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
