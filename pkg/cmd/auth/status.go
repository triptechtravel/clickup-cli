package auth

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/auth"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdStatus returns the "auth status" command.
func NewCmdStatus(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long:  "Display information about the current authentication state.",
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return statusRun(f)
		},
	}

	return cmd
}

func statusRun(f *cmdutil.Factory) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	// Retrieve and validate the stored token.
	token, err := auth.GetToken()
	if err != nil {
		return fmt.Errorf("not authenticated: %w", err)
	}

	user, err := auth.ValidateToken(token)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	method := auth.GetAuthMethod()

	cfg, err := f.Config()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	workspace := cfg.Workspace
	if workspace == "" {
		workspace = "(none)"
	}

	fmt.Fprintf(ios.Out, "%s Logged in to ClickUp\n", cs.Green("!"))
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Username:"), user.Username)
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Email:"), user.Email)
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Auth method:"), method)
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Workspace:"), workspace)

	return nil
}
