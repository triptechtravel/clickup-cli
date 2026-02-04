package task

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdTask returns the top-level "task" command that groups view, list, create, and edit.
func NewCmdTask(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task <command>",
		Short: "Manage ClickUp tasks",
		Long:  "View, list, create, edit, and search tasks. Track time, manage dependencies and checklists.",
	}

	cmd.AddCommand(NewCmdView(f))
	cmd.AddCommand(NewCmdList(f))
	cmd.AddCommand(NewCmdCreate(f))
	cmd.AddCommand(NewCmdEdit(f))
	cmd.AddCommand(NewCmdDelete(f))
	cmd.AddCommand(NewCmdSearch(f))
	cmd.AddCommand(NewCmdActivity(f))
	cmd.AddCommand(NewCmdTime(f))
	cmd.AddCommand(NewCmdDependency(f))
	cmd.AddCommand(NewCmdChecklist(f))
	cmd.AddCommand(NewCmdRecent(f))

	return cmd
}
