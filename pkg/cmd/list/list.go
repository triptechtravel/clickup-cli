package list

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdList returns the list parent command.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Manage lists",
		Long:  "List ClickUp lists in your spaces or folders.",
	}

	cmd.AddCommand(NewCmdListList(f))
	cmd.AddCommand(NewCmdListSelect(f))

	return cmd
}
