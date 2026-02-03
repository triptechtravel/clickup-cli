package link

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type syncOptions struct {
	factory  *cmdutil.Factory
	taskID   string
	repo     string
	prNumber int
}

// NewCmdLinkSync returns the "link sync" command.
func NewCmdLinkSync(f *cmdutil.Factory) *cobra.Command {
	opts := &syncOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "sync [PR-NUMBER]",
		Short: "Sync ClickUp task info to a GitHub PR",
		Long: `Update a GitHub pull request with information from the linked ClickUp task.

Adds the ClickUp task URL and status to the PR body, and updates the task
description (or configured custom field) with a link to the PR.

The task ID is auto-detected from the branch name, or specified with --task.`,
		Example: `  # Sync current branch's PR with the detected task
  clickup link sync

  # Sync a specific PR with a specific task
  clickup link sync 1109 --repo owner/repo --task 86d1rn980`,
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
			return syncRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.taskID, "task", "", "ClickUp task ID (auto-detected from branch if not set)")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "GitHub repository (owner/repo)")

	return cmd
}

func syncRun(opts *syncOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID.
	resolved, err := resolveTask(opts.factory, opts.taskID)
	if err != nil {
		return err
	}
	taskID := resolved.TaskID

	fmt.Fprintf(ios.ErrOut, "Syncing task %s with GitHub PR...\n", cs.Bold(taskID))

	// Fetch task details from ClickUp.
	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	task, _, err := client.Clickup.Tasks.GetTask(ctx, taskID, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch task: %w", err)
	}

	taskURL := fmt.Sprintf("https://app.clickup.com/t/%s", taskID)
	taskName := task.Name
	taskStatus := task.Status.Status

	var assigneeNames []string
	for _, a := range task.Assignees {
		assigneeNames = append(assigneeNames, a.Username)
	}

	priority := ""
	if task.Priority.Priority != "" {
		priority = task.Priority.Priority
	}

	// Fetch the PR details.
	var pr *ghPR
	if opts.prNumber > 0 {
		pr, err = fetchPR(opts.prNumber, opts.repo)
	} else {
		pr, err = fetchCurrentPR()
	}
	if err != nil {
		return err
	}

	// Build the ClickUp info block for the PR body.
	clickupBlock := buildClickUpBlock(taskURL, taskName, taskStatus, priority, assigneeNames)

	// Update the PR body.
	newBody := upsertClickUpBlock(pr.Body, clickupBlock)
	if newBody != pr.Body {
		if err := updatePRBody(pr.Number, opts.repo, newBody); err != nil {
			return fmt.Errorf("failed to update PR body: %w", err)
		}
		fmt.Fprintf(ios.Out, "%s Updated PR #%d body with ClickUp task info\n",
			cs.Green("!"), pr.Number)
	} else {
		fmt.Fprintf(ios.Out, "PR #%d body already up to date\n", pr.Number)
	}

	// Upsert link on ClickUp task (description or custom field).
	repoSlug := opts.repo
	if repoSlug == "" {
		repoSlug = inferRepoFromURL(pr.URL)
	}

	entry := buildPREntry(repoSlug, pr.Number, pr.Title, pr.URL)
	if err := upsertLink(opts.factory, taskID, entry); err != nil {
		return err
	}
	fmt.Fprintf(ios.Out, "%s Linked PR #%d to task %s\n",
		cs.Green("!"), pr.Number, cs.Bold(taskID))

	return nil
}

const clickupBlockStart = "<!-- clickup-cli:start -->"
const clickupBlockEnd = "<!-- clickup-cli:end -->"

func buildClickUpBlock(taskURL, taskName, status, priority string, assignees []string) string {
	var sb strings.Builder
	sb.WriteString(clickupBlockStart)
	sb.WriteString("\n")
	sb.WriteString("## ClickUp Task\n\n")
	sb.WriteString(fmt.Sprintf("| | |\n|---|---|\n"))
	sb.WriteString(fmt.Sprintf("| **Task** | [%s](%s) |\n", taskName, taskURL))
	sb.WriteString(fmt.Sprintf("| **Status** | %s |\n", status))
	if priority != "" {
		sb.WriteString(fmt.Sprintf("| **Priority** | %s |\n", priority))
	}
	if len(assignees) > 0 {
		sb.WriteString(fmt.Sprintf("| **Assignees** | %s |\n", strings.Join(assignees, ", ")))
	}
	sb.WriteString("\n")
	sb.WriteString(clickupBlockEnd)
	return sb.String()
}

func upsertClickUpBlock(body, block string) string {
	startIdx := strings.Index(body, clickupBlockStart)
	endIdx := strings.Index(body, clickupBlockEnd)

	if startIdx >= 0 && endIdx >= 0 {
		return body[:startIdx] + block + body[endIdx+len(clickupBlockEnd):]
	}

	if body == "" {
		return block
	}
	return block + "\n\n" + body
}

func updatePRBody(number int, repo, body string) error {
	args := []string{"pr", "edit", strconv.Itoa(number), "--body", body}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	cmd := exec.Command("gh", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
