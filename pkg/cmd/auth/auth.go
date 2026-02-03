package auth

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdAuth returns the top-level "auth" command that groups login, logout, and status.
func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Authenticate with ClickUp",
		Long:  "Manage authentication state for the ClickUp CLI.",
	}

	cmd.AddCommand(NewCmdLogin(f))
	cmd.AddCommand(NewCmdLogout(f))
	cmd.AddCommand(NewCmdStatus(f))

	return cmd
}
