package goal

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdGoal returns the goal parent command.
func NewCmdGoal(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goal",
		Short: "Manage goals",
		Long:  "List, view, create, and delete ClickUp goals.",
	}

	cmd.AddCommand(NewCmdGoalList(f))
	cmd.AddCommand(NewCmdGoalView(f))
	cmd.AddCommand(NewCmdGoalCreate(f))
	cmd.AddCommand(NewCmdGoalDelete(f))

	return cmd
}
