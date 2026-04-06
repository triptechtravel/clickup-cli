package doc

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdDoc returns the top-level "doc" command that groups Docs subcommands.
func NewCmdDoc(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doc <command>",
		Short: "Manage ClickUp Docs",
		Long:  "List, view, and create ClickUp Docs and their pages.",
	}

	cmd.AddCommand(NewCmdList(f))
	cmd.AddCommand(NewCmdView(f))
	cmd.AddCommand(NewCmdCreate(f))
	cmd.AddCommand(NewCmdPage(f))

	return cmd
}
