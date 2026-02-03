package link

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type commitOptions struct {
	factory *cmdutil.Factory
	sha     string
	taskID  string
	repo    string
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
	cmd.Flags().StringVar(&opts.repo, "repo", "", "GitHub repository (owner/repo) for the commit URL")

	return cmd
}

func commitRun(opts *commitOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID.
	resolved, err := resolveTask(opts.factory, opts.taskID)
	if err != nil {
		return err
	}
	taskID := resolved.TaskID

	// Determine repo slug for commit URL.
	repoSlug := opts.repo
	if repoSlug == "" && resolved.GitCtx != nil {
		repoSlug = fmt.Sprintf("%s/%s", resolved.GitCtx.RepoOwner, resolved.GitCtx.RepoName)
	}
	if repoSlug == "" {
		return fmt.Errorf("could not detect repository. Use --repo to specify (e.g., --repo owner/repo)")
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

	// Build rich text comment blocks.
	commitURL := fmt.Sprintf("https://github.com/%s/commit/%s", repoSlug, fullSHA)
	blocks := []commentBlock{
		{Text: "\xf0\x9f\x94\x97 "},
		{Text: "Commit linked", Attributes: map[string]interface{}{"bold": true}},
		{Text: ": "},
		{Text: shortSHA, Attributes: map[string]interface{}{"code": true, "link": commitURL}},
		{Text: " - " + commitMessage},
		{Text: "\n"},
	}

	// Post the comment.
	if err := postRichComment(opts.factory, taskID, blocks); err != nil {
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
