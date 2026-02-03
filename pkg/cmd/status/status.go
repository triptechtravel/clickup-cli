package status

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdStatus returns the top-level "status" command that groups set and list subcommands.
func NewCmdStatus(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <command>",
		Short: "Manage task statuses",
		Long:  "Set task statuses and list available statuses for a space.",
	}

	cmd.AddCommand(NewCmdSet(f))
	cmd.AddCommand(NewCmdList(f))

	return cmd
}
