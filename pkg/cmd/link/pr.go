package link

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type prOptions struct {
	factory  *cmdutil.Factory
	prNumber int
	taskID   string
	repo     string
}

// NewCmdLinkPR returns the "link pr" command.
func NewCmdLinkPR(f *cmdutil.Factory) *cobra.Command {
	opts := &prOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "pr [NUMBER]",
		Short: "Link a GitHub PR to a ClickUp task",
		Long: `Link a GitHub pull request to a ClickUp task.

Updates the task description (or a configured custom field) with a link to
the PR. Running the command again updates the existing entry rather than
creating duplicates.

If NUMBER is not provided, the current PR is detected using the GitHub CLI (gh).
The ClickUp task ID is auto-detected from the current git branch name,
or can be specified explicitly with --task.`,
		Example: `  # Link the current branch's PR to the detected task
  clickup link pr

  # Link a specific PR by number
  clickup link pr 42

  # Link a PR from another repo to a specific task
  clickup link pr 1109 --repo owner/repo --task 86d1rn980`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 1 {
				n, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid PR number %q: %w", args[0], err)
				}
				opts.prNumber = n
			}
			return prRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.taskID, "task", "", "ClickUp task ID (auto-detected from branch if not set)")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "GitHub repository (owner/repo) for the PR")

	return cmd
}

func prRun(opts *prOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID.
	resolved, err := resolveTask(opts.factory, opts.taskID)
	if err != nil {
		return err
	}
	taskID := resolved.TaskID

	var repoSlug string
	if resolved.GitCtx != nil {
		repoSlug = fmt.Sprintf("%s/%s", resolved.GitCtx.RepoOwner, resolved.GitCtx.RepoName)
	}
	if opts.repo != "" {
		repoSlug = opts.repo
	}

	// Resolve PR info.
	var pr *ghPR
	if opts.prNumber > 0 {
		pr, err = fetchPR(opts.prNumber, opts.repo)
	} else {
		// Try current branch first, then search by task ID if --task was given.
		pr, err = fetchCurrentPR()
		if err != nil && opts.taskID != "" {
			pr, err = fetchPRForTaskID(taskID, opts.repo)
		}
	}
	if err != nil {
		return err
	}

	// Infer repo slug from PR URL if we don't have it yet.
	if repoSlug == "" {
		repoSlug = inferRepoFromURL(pr.URL)
	}

	entry := buildPREntry(repoSlug, pr.Number, pr.Title, pr.URL)

	if err := upsertLink(opts.factory, taskID, entry); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "%s Linked PR #%d to task %s\n",
		cs.Green("!"), pr.Number, cs.Bold(taskID))

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup link sync\n", cs.Gray("Sync:"))
	fmt.Fprintf(ios.Out, "  %s  clickup status set <status> %s\n", cs.Gray("Status:"), taskID)

	return nil
}
