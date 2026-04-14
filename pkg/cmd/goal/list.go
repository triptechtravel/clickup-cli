package goal

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdGoalList returns the goal list command.
func NewCmdGoalList(f *cmdutil.Factory) *cobra.Command {
	var (
		includeCompleted bool
		jsonFlags        cmdutil.JSONFlags
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List goals in the workspace",
		Long:  "List all goals in your ClickUp workspace.",
		Example: `  # List goals
  clickup goal list

  # Include completed goals
  clickup goal list --completed

  # Output as JSON
  clickup goal list --json`,
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

			resp, err := apiv2.GetGoals(context.Background(), client, teamID, apiv2.GetGoalsParams{
				IncludeCompleted: includeCompleted,
			})
			if err != nil {
				return fmt.Errorf("failed to fetch goals: %w", err)
			}

			if jsonFlags.WantsJSON() {
				return jsonFlags.OutputJSON(f.IOStreams.Out, resp.Goals)
			}

			if len(resp.Goals) == 0 {
				fmt.Fprintln(f.IOStreams.Out, "No goals found.")
				return nil
			}

			tp := tableprinter.New(f.IOStreams)

			for _, g := range resp.Goals {
				tp.AddField(g.Name)
				tp.AddField(g.ID)
				tp.AddField(fmt.Sprintf("%d%%", g.PercentCompleted))
				tp.AddField(g.DueDate)
				tp.EndRow()
			}

			if err := tp.Render(); err != nil {
				return err
			}

			cs := f.IOStreams.ColorScheme()
			fmt.Fprintln(f.IOStreams.Out)
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("---"))
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("Quick actions:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup goal view <id>\n", cs.Gray("View:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup goal list --json\n", cs.Gray("JSON:"))

			return nil
		},
	}

	cmd.Flags().BoolVar(&includeCompleted, "completed", false, "Include completed goals")
	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}
