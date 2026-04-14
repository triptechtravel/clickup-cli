package view

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdViewTasks returns the view tasks command.
func NewCmdViewTasks(f *cmdutil.Factory) *cobra.Command {
	var (
		page      int
		jsonFlags cmdutil.JSONFlags
	)

	cmd := &cobra.Command{
		Use:   "tasks <view-id>",
		Short: "List tasks in a view",
		Long:  "List all tasks visible in a ClickUp view.",
		Example: `  # List tasks in a view
  clickup view tasks 3v-abc123

  # Page through results
  clickup view tasks 3v-abc123 --page 1

  # Output as JSON
  clickup view tasks 3v-abc123 --json`,
		Args:    cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			viewID := args[0]

			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			resp, err := apiv2.GetViewTasks(context.Background(), client, viewID, apiv2.GetViewTasksParams{
				Page: page,
			})
			if err != nil {
				return fmt.Errorf("failed to fetch view tasks: %w", err)
			}

			if jsonFlags.WantsJSON() {
				return jsonFlags.OutputJSON(f.IOStreams.Out, resp.Tasks)
			}

			if len(resp.Tasks) == 0 {
				fmt.Fprintln(f.IOStreams.Out, "No tasks found.")
				return nil
			}

			tp := tableprinter.New(f.IOStreams)

			for _, t := range resp.Tasks {
				id := ""
				if t.ID != nil {
					id = *t.ID
				}
				name := ""
				if t.Name != nil {
					name = *t.Name
				}
				tp.AddField(id)
				tp.AddField(name)
				tp.EndRow()
			}

			if err := tp.Render(); err != nil {
				return err
			}

			if !resp.LastPage {
				cs := f.IOStreams.ColorScheme()
				fmt.Fprintln(f.IOStreams.Out)
				fmt.Fprintf(f.IOStreams.Out, "%s More results available. Use --page %d\n", cs.Gray("..."), page+1)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&page, "page", 0, "Page number (0-indexed)")
	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}
