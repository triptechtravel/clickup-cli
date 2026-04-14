package webhook

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdWebhookList returns the webhook list command.
func NewCmdWebhookList(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhooks",
		Long:  "List all webhooks in your ClickUp workspace.",
		Example: `  # List webhooks
  clickup webhook list

  # Output as JSON
  clickup webhook list --json`,
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			resp, err := apiv2.GetWebhooks(context.Background(), client, teamID)
			if err != nil {
				return fmt.Errorf("failed to fetch webhooks: %w", err)
			}

			if jsonFlags.WantsJSON() {
				return jsonFlags.OutputJSON(f.IOStreams.Out, resp.Webhooks)
			}

			if len(resp.Webhooks) == 0 {
				fmt.Fprintln(f.IOStreams.Out, "No webhooks found.")
				return nil
			}

			tp := tableprinter.New(f.IOStreams)

			for _, wh := range resp.Webhooks {
				tp.AddField(wh.ID)
				tp.AddField(wh.Endpoint)
				tp.AddField(formatEvents(wh.Events))
				tp.AddField(formatHealth(wh.Health))
				tp.EndRow()
			}

			if err := tp.Render(); err != nil {
				return err
			}

			cs := f.IOStreams.ColorScheme()
			fmt.Fprintln(f.IOStreams.Out)
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("---"))
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("Quick actions:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup webhook create --endpoint <url> --events <events...>\n", cs.Gray("Create:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup webhook delete <id>\n", cs.Gray("Delete:"))

			return nil
		},
	}

	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}

func formatEvents(events []any) string {
	if len(events) == 0 {
		return ""
	}
	strs := make([]string, 0, len(events))
	for _, e := range events {
		strs = append(strs, fmt.Sprintf("%v", e))
	}
	return strings.Join(strs, ", ")
}

func formatHealth(health any) string {
	if health == nil {
		return ""
	}
	return fmt.Sprintf("%v", health)
}
