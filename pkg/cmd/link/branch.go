package link

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
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
		Long: `Link the current git branch to a ClickUp task by posting a comment.

The ClickUp task ID is auto-detected from the current git branch name,
or can be specified explicitly with --task.`,
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
	var taskID string
	gitCtx, err := opts.factory.GitContext()
	if err != nil && opts.taskID == "" {
		return fmt.Errorf("could not detect git context: %w\n\nTip: use --task to specify the task ID explicitly", err)
	}

	if opts.taskID != "" {
		taskID = opts.taskID
		fmt.Fprintf(ios.ErrOut, "Using task %s\n", cs.Bold(taskID))
	} else {
		if gitCtx.TaskID == nil {
			return fmt.Errorf("%s\n\nTip: use --task to specify the task ID explicitly", git.BranchNamingSuggestion(gitCtx.Branch))
		}
		taskID = gitCtx.TaskID.ID
		fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n", cs.Bold(taskID), cs.Cyan(gitCtx.Branch))
	}

	// Build rich text comment blocks.
	repoSlug := fmt.Sprintf("%s/%s", gitCtx.RepoOwner, gitCtx.RepoName)
	blocks := []commentBlock{
		{Text: "\xf0\x9f\x94\x97 "},
		{Text: "Branch linked", Attributes: map[string]interface{}{"bold": true}},
		{Text: ": "},
		{Text: gitCtx.Branch, Attributes: map[string]interface{}{"code": true}},
		{Text: " in " + repoSlug},
		{Text: "\n"},
	}

	// Post the comment.
	if err := postRichComment(opts.factory, taskID, blocks); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "%s Linked branch %s to task %s\n",
		cs.Green("!"), cs.Cyan(gitCtx.Branch), cs.Bold(taskID))
	return nil
}
