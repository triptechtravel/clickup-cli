package list

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdList returns the "list" parent command.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Manage lists",
		Long:  "List and browse ClickUp lists within spaces and folders.",
	}

	cmd.AddCommand(NewCmdListLs(f))

	return cmd
}
