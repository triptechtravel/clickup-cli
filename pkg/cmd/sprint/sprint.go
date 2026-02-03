package sprint

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdSprint returns the sprint parent command.
func NewCmdSprint(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sprint",
		Short: "Manage sprints",
		Long:  "List and view sprints in your ClickUp workspace.",
	}

	cmd.AddCommand(NewCmdSprintList(f))
	cmd.AddCommand(NewCmdSprintCurrent(f))

	return cmd
}
