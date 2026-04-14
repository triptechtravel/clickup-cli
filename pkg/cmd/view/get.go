package view

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdViewGet returns the view get command.
func NewCmdViewGet(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags

	cmd := &cobra.Command{
		Use:   "get <view-id>",
		Short: "Get a view",
		Long: `Get detailed information about a ClickUp view.

Output is always JSON because the response is a union type that cannot
be rendered as a table.`,
		Example: `  # Get a view
  clickup view get 3v-abc123`,
		Args:    cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			viewID := args[0]

			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			resp, err := apiv2.GetView(context.Background(), client, viewID)
			if err != nil {
				return fmt.Errorf("failed to fetch view: %w", err)
			}

			// Response is a union type — always output as JSON.
			return jsonFlags.OutputJSON(f.IOStreams.Out, resp)
		},
	}

	// Keep JSON flags for --jq and --template support; output is always JSON.
	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}
