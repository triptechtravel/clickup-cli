package link

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdLink returns the top-level "link" command that groups pr, branch, and commit subcommands.
func NewCmdLink(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link <command>",
		Short: "Link GitHub objects to ClickUp tasks",
		Long:  "Link pull requests, branches, and commits to ClickUp tasks by posting a comment on the task.",
	}

	cmd.AddCommand(NewCmdLinkPR(f))
	cmd.AddCommand(NewCmdLinkBranch(f))
	cmd.AddCommand(NewCmdLinkCommit(f))

	return cmd
}
