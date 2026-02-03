package comment

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdComment returns the top-level "comment" command that groups add, list, and edit.
func NewCmdComment(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment <command>",
		Short: "Manage comments on ClickUp tasks",
		Long:  "Add, list, edit, and delete comments on ClickUp tasks.",
	}

	cmd.AddCommand(NewCmdAdd(f))
	cmd.AddCommand(NewCmdList(f))
	cmd.AddCommand(NewCmdEdit(f))
	cmd.AddCommand(NewCmdDelete(f))

	return cmd
}
