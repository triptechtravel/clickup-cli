package folder

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdFolder returns the folder parent command.
func NewCmdFolder(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folder",
		Short: "Manage folders",
		Long:  "List folders in your ClickUp spaces.",
	}

	cmd.AddCommand(NewCmdFolderList(f))
	cmd.AddCommand(NewCmdFolderSelect(f))

	return cmd
}
