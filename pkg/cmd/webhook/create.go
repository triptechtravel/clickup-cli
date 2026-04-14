package webhook

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	clickupv2 "github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdWebhookCreate returns the webhook create command.
func NewCmdWebhookCreate(f *cmdutil.Factory) *cobra.Command {
	var (
		endpoint string
		events   []string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook",
		Long:  "Create a new webhook in your ClickUp workspace.",
		Example: `  # Create a webhook for all events
  clickup webhook create --endpoint https://example.com/hook --events "*"

  # Create a webhook for specific events
  clickup webhook create --endpoint https://example.com/hook --events taskCreated --events taskUpdated`,
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if endpoint == "" {
				return fmt.Errorf("--endpoint is required")
			}
			if len(events) == 0 {
				return fmt.Errorf("--events is required")
			}

			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			cfg, err := f.Config()
			if err != nil {
				return err
			}

			teamID := cfg.Workspace
			if teamID == "" {
				return fmt.Errorf("no workspace configured. Run 'clickup auth' first")
			}

			req := &clickupv2.CreateWebhookJSONRequest{
				Endpoint: endpoint,
				Events:   events,
			}

			resp, err := apiv2.CreateWebhook(context.Background(), client, teamID, req)
			if err != nil {
				return fmt.Errorf("failed to create webhook: %w", err)
			}

			cs := f.IOStreams.ColorScheme()
			fmt.Fprintf(f.IOStreams.Out, "%s Webhook created (ID: %s)\n", cs.Green("!"), cs.Bold(resp.ID))

			return nil
		},
	}

	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Webhook endpoint URL (required)")
	cmd.Flags().StringSliceVar(&events, "events", nil, "Event types to subscribe to (required)")

	return cmd
}
