package attachment

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdAttachment returns the top-level "attachment" command that groups list and add.
func NewCmdAttachment(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachment <command>",
		Short: "Manage attachments on ClickUp tasks",
		Long:  "List and upload attachments on ClickUp tasks.",
	}

	cmd.AddCommand(NewCmdList(f))
	cmd.AddCommand(NewCmdAdd(f))

	return cmd
}
