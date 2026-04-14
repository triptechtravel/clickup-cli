package webhook

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdWebhook returns the webhook parent command.
func NewCmdWebhook(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage webhooks",
		Long:  "List, create, and delete ClickUp webhooks.",
	}

	cmd.AddCommand(NewCmdWebhookList(f))
	cmd.AddCommand(NewCmdWebhookCreate(f))
	cmd.AddCommand(NewCmdWebhookDelete(f))

	return cmd
}
