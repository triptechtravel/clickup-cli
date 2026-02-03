package link

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type branchOptions struct {
	factory *cmdutil.Factory
	taskID  string
}

// NewCmdLinkBranch returns the "link branch" command.
func NewCmdLinkBranch(f *cmdutil.Factory) *cobra.Command {
	opts := &branchOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Link the current git branch to a ClickUp task",
		Long: `Link the current git branch to a ClickUp task.

Updates the task description (or a configured custom field) with a reference
to the branch. Running the command again updates the existing entry rather
than creating duplicates.

The ClickUp task ID is auto-detected from the current git branch name,
or can be specified explicitly with --task.`,
		Example: `  # Link current branch to auto-detected task
  clickup link branch

  # Link to a specific task
  clickup link branch --task CU-abc123`,
		Args:              cobra.NoArgs,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return branchRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.taskID, "task", "", "ClickUp task ID (auto-detected from branch if not set)")

	return cmd
}

func branchRun(opts *branchOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID.
	resolved, err := resolveTask(opts.factory, opts.taskID)
	if err != nil {
		return err
	}
	taskID := resolved.TaskID

	if resolved.GitCtx == nil {
		return fmt.Errorf("could not detect git context: not inside a git repository")
	}
	gitCtx := resolved.GitCtx

	// Build link entry (markdown format for ClickUp rich rendering).
	repoSlug := fmt.Sprintf("%s/%s", gitCtx.RepoOwner, gitCtx.RepoName)
	entry := linkEntry{
		Prefix: fmt.Sprintf("`%s` in %s", gitCtx.Branch, repoSlug),
		Line:   fmt.Sprintf("Branch: `%s` in %s", gitCtx.Branch, repoSlug),
	}

	if err := upsertLink(opts.factory, taskID, entry); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "%s Linked branch %s to task %s\n",
		cs.Green("!"), cs.Cyan(gitCtx.Branch), cs.Bold(taskID))
	return nil
}
