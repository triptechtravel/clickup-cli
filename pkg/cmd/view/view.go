package view

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdView returns the view parent command.
func NewCmdView(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Manage views",
		Long:  "List views and view tasks in ClickUp.",
	}

	cmd.AddCommand(NewCmdViewList(f))
	cmd.AddCommand(NewCmdViewGet(f))
	cmd.AddCommand(NewCmdViewTasks(f))

	return cmd
}
