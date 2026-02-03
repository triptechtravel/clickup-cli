package version

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/build"
)

// NewCmdVersion returns the version command.
func NewCmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of clickup CLI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "clickup version %s (%s)\nbuilt %s\n",
				build.Version, build.Commit, build.Date)
		},
	}
}
