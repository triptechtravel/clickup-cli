package auth

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/auth"
	"github.com/triptechtravel/clickup-cli/internal/config"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdLogout returns the "auth logout" command.
func NewCmdLogout(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of ClickUp",
		Long: `Remove stored authentication credentials for the ClickUp CLI.

Also deletes the local task index, which mirrors every task name and
description in the workspace to disk. Revoking access should not leave that
readable on the machine.`,
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

	// The cached index holds the workspace's task names and descriptions in
	// plaintext. Clearing the token while leaving that behind revokes access to
	// the API and to nothing that was already copied off it.
	cacheDir := config.CacheDir()
	if err := os.RemoveAll(cacheDir); err != nil {
		fmt.Fprintf(ios.ErrOut, "warning: could not remove cached task index at %s: %v\n", cacheDir, err)
	}

	fmt.Fprintf(ios.Out, "%s Logged out of ClickUp, and removed the cached task index\n", cs.Green("!"))
	return nil
}
