package auth

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/auth"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdLogout returns the "auth logout" command.
func NewCmdLogout(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of ClickUp",
		Long:  "Remove stored authentication credentials for the ClickUp CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return logoutRun(f)
		},
	}

	return cmd
}

func logoutRun(f *cmdutil.Factory) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	if err := auth.ClearToken(); err != nil {
		return fmt.Errorf("failed to clear credentials: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Logged out of ClickUp\n", cs.Green("!"))
	return nil
}
