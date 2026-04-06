package doc

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdPage returns the "doc page" subcommand group.
func NewCmdPage(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page <command>",
		Short: "Manage pages within a ClickUp Doc",
		Long:  "List, view, create, and edit pages within a ClickUp Doc.",
	}

	cmd.AddCommand(NewCmdPageList(f))
	cmd.AddCommand(NewCmdPageView(f))
	cmd.AddCommand(NewCmdPageCreate(f))
	cmd.AddCommand(NewCmdPageEdit(f))

	return cmd
}
