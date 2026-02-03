package link

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type branchOptions struct {
	factory *cmdutil.Factory
}

// NewCmdLinkBranch returns the "link branch" command.
func NewCmdLinkBranch(f *cmdutil.Factory) *cobra.Command {
	opts := &branchOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Link the current git branch to a ClickUp task",
		Long: `Link the current git branch to a ClickUp task by posting a comment.

The ClickUp task ID is auto-detected from the current git branch name.`,
		Args:              cobra.NoArgs,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return branchRun(opts)
		},
	}

	return cmd
}

func branchRun(opts *branchOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID from git branch.
	gitCtx, err := opts.factory.GitContext()
	if err != nil {
		return fmt.Errorf("could not detect git context: %w\n\n%s", err,
			"Tip: run this command from inside a git repository")
	}
	if gitCtx.TaskID == nil {
		return fmt.Errorf("%s", git.BranchNamingSuggestion(gitCtx.Branch))
	}
	taskID := gitCtx.TaskID.ID
	fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n", cs.Bold(taskID), cs.Cyan(gitCtx.Branch))

	// Build the comment text.
	repoSlug := fmt.Sprintf("%s/%s", gitCtx.RepoOwner, gitCtx.RepoName)
	commentText := fmt.Sprintf(
		"\xf0\x9f\x94\x97 **Branch linked**: `%s` in %s",
		gitCtx.Branch, repoSlug,
	)

	// Post the comment.
	if err := postComment(opts.factory, taskID, commentText); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "%s Linked branch %s to task %s\n",
		cs.Green("!"), cs.Cyan(gitCtx.Branch), cs.Bold(taskID))
	return nil
}
