package cmdutil

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/auth"
)

// NeedsAuth returns a pre-run function that validates authentication before command execution.
func NeedsAuth(f *Factory) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		token, err := auth.GetToken()
		if err != nil || token == "" {
			return &AuthError{}
		}
		return nil
	}
}
