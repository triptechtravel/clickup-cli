package link

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type commitOptions struct {
	factory *cmdutil.Factory
	sha     string
	taskID  string
}

// NewCmdLinkCommit returns the "link commit" command.
func NewCmdLinkCommit(f *cmdutil.Factory) *cobra.Command {
	opts := &commitOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "commit [SHA]",
		Short: "Link a git commit to a ClickUp task",
		Long: `Link a git commit to a ClickUp task by posting a comment.

If SHA is not provided, the HEAD commit is used.
The ClickUp task ID is auto-detected from the current git branch name,
or can be specified explicitly with --task.`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 1 {
				opts.sha = args[0]
			}
			return commitRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.taskID, "task", "", "ClickUp task ID (auto-detected from branch if not set)")

	return cmd
}

func commitRun(opts *commitOptions) error {
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

	// Resolve commit SHA and message.
	gitClient := opts.factory.GitClient()

	var fullSHA string
	var commitMessage string

	if opts.sha != "" {
		// Specific SHA provided - resolve full SHA and message via git log.
		resolvedSHA, err := resolveCommitSHA(opts.sha)
		if err != nil {
			return fmt.Errorf("could not resolve commit %q: %w", opts.sha, err)
		}
		fullSHA = resolvedSHA

		msg, err := getCommitMessage(opts.sha)
		if err != nil {
			return fmt.Errorf("could not get commit message for %q: %w", opts.sha, err)
		}
		commitMessage = msg
	} else {
		// Use HEAD commit.
		sha, err := gitClient.LatestCommitSHA()
		if err != nil {
			return fmt.Errorf("could not get latest commit SHA: %w", err)
		}
		fullSHA = sha

		msg, err := gitClient.LatestCommitMessage()
		if err != nil {
			return fmt.Errorf("could not get latest commit message: %w", err)
		}
		commitMessage = msg
	}

	shortSHA := fullSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	// Build the comment text.
	commitURL := fmt.Sprintf("https://github.com/%s/%s/commit/%s",
		gitCtx.RepoOwner, gitCtx.RepoName, fullSHA)
	commentText := fmt.Sprintf(
		"\xf0\x9f\x94\x97 **Commit linked**: [`%s`](%s) - %s",
		shortSHA, commitURL, commitMessage,
	)

	// Post the comment.
	if err := postComment(opts.factory, taskID, commentText); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "%s Linked commit %s to task %s\n",
		cs.Green("!"), cs.Cyan(shortSHA), cs.Bold(taskID))
	return nil
}

// resolveCommitSHA resolves a commit reference to its full SHA.
func resolveCommitSHA(ref string) (string, error) {
	cmd := exec.Command("git", "rev-parse", ref)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// getCommitMessage returns the subject line of a commit by its reference.
func getCommitMessage(ref string) (string, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%s", ref)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
