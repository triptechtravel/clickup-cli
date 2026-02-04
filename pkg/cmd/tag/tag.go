package tag

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdTag returns the top-level "tag" command.
func NewCmdTag(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag <command>",
		Short: "Manage space tags",
		Long:  "View and manage tags available in your ClickUp spaces.",
	}

	cmd.AddCommand(NewCmdTagList(f))

	return cmd
}
