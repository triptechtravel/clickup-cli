package goal

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdGoalView returns the goal view command.
func NewCmdGoalView(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags

	cmd := &cobra.Command{
		Use:   "view <goal-id>",
		Short: "View a goal",
		Long:  "View detailed information about a ClickUp goal.",
		Example: `  # View a goal
  clickup goal view e53a33d0-2eb2-4664-a4b3-5e1b0df0e912

  # View as JSON
  clickup goal view e53a33d0-2eb2-4664-a4b3-5e1b0df0e912 --json`,
		Args:    cobra.ExactArgs(1),
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			goalID := args[0]

			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			resp, err := apiv2.GetGoal(context.Background(), client, goalID)
			if err != nil {
				return fmt.Errorf("failed to fetch goal: %w", err)
			}

			goal := resp.Goal

			if jsonFlags.WantsJSON() {
				return jsonFlags.OutputJSON(f.IOStreams.Out, goal)
			}

			cs := f.IOStreams.ColorScheme()
			out := f.IOStreams.Out

			fmt.Fprintf(out, "%s %s\n", cs.Bold("Name:"), goal.Name)
			fmt.Fprintf(out, "%s %s\n", cs.Bold("ID:"), goal.ID)
			fmt.Fprintf(out, "%s %s\n", cs.Bold("Description:"), goal.Description)
			fmt.Fprintf(out, "%s %s\n", cs.Bold("Due Date:"), goal.DueDate)
			fmt.Fprintf(out, "%s %s\n", cs.Bold("Color:"), goal.Color)
			fmt.Fprintf(out, "%s %v\n", cs.Bold("Private:"), goal.Private)
			fmt.Fprintf(out, "%s %v\n", cs.Bold("Archived:"), goal.Archived)
			fmt.Fprintf(out, "%s %s\n", cs.Bold("Date Created:"), goal.DateCreated)

			return nil
		},
	}

	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}
