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
		Long: `Link pull requests, branches, and commits to ClickUp tasks.

By default, links are stored in a managed section of the task description
using ClickUp's markdown_description API field, so they render as rich text
with clickable links, bold formatting, and code blocks directly in the
ClickUp UI. Running the same command again updates the existing entry rather
than creating duplicates.

Optionally, configure a custom field for link storage:
  Set 'link_field' in ~/.config/clickup/config.yml to the name of a
  URL or text custom field in your workspace (e.g., link_field: "link_url").
  Per-directory overrides are also supported in directory_defaults.`,
	}

	cmd.AddCommand(NewCmdLinkPR(f))
	cmd.AddCommand(NewCmdLinkBranch(f))
	cmd.AddCommand(NewCmdLinkCommit(f))
	cmd.AddCommand(NewCmdLinkSync(f))

	return cmd
}
