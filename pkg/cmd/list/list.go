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
		Long:  "List and select ClickUp lists in a folder or space.",
	}

	cmd.AddCommand(NewCmdListList(f))
	cmd.AddCommand(NewCmdListSelect(f))
	cmd.AddCommand(NewCmdListCreate(f))
	cmd.AddCommand(NewCmdListDelete(f))

	return cmd
}
