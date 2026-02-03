package member

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdMember returns the member parent command.
func NewCmdMember(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage workspace members",
		Long:  "List and look up members of your ClickUp workspace.",
	}

	cmd.AddCommand(NewCmdMemberList(f))

	return cmd
}
