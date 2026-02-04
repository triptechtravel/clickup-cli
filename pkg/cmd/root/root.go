package root

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/auth"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/comment"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/completion"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/field"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/inbox"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/link"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/member"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/space"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/sprint"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/status"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/tag"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/task"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/version"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdRoot creates the root command for the clickup CLI.
func NewCmdRoot(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clickup",
		Short: "ClickUp CLI - manage tasks from the command line",
		Long: `Work with ClickUp tasks, comments, and sprints from your terminal.

Integrates with git to auto-detect tasks from branch names.
Links GitHub PRs, branches, and commits to ClickUp tasks.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Core commands
	cmd.AddCommand(auth.NewCmdAuth(f))
	cmd.AddCommand(task.NewCmdTask(f))
	cmd.AddCommand(comment.NewCmdComment(f))
	cmd.AddCommand(status.NewCmdStatus(f))

	// Workflow commands
	cmd.AddCommand(link.NewCmdLink(f))
	cmd.AddCommand(sprint.NewCmdSprint(f))
	cmd.AddCommand(space.NewCmdSpace(f))
	cmd.AddCommand(field.NewCmdField(f))
	cmd.AddCommand(tag.NewCmdTag(f))

	// Workspace
	cmd.AddCommand(member.NewCmdMember(f))

	// Inbox
	cmd.AddCommand(inbox.NewCmdInbox(f))

	// Utility commands
	cmd.AddCommand(version.NewCmdVersion())
	cmd.AddCommand(completion.NewCmdCompletion(cmd))

	return cmd
}
