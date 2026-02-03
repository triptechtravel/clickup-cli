package space

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdSpace returns the space parent command.
func NewCmdSpace(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "space",
		Short: "Manage spaces",
		Long:  "List and select ClickUp spaces in your workspace.",
	}

	cmd.AddCommand(NewCmdSpaceList(f))
	cmd.AddCommand(NewCmdSpaceSelect(f))

	return cmd
}
