package link

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
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

Adds the ClickUp task URL and status to the PR body, and posts a link
comment on the ClickUp task if one doesn't already exist.

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

type ghPRDetail struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	URL    string `json:"url"`
}

func syncRun(opts *syncOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID.
	var taskID string
	if opts.taskID != "" {
		taskID = opts.taskID
	} else {
		gitCtx, err := opts.factory.GitContext()
		if err != nil {
			return fmt.Errorf("could not detect git context: %w\n\nTip: use --task to specify the task ID explicitly", err)
		}
		if gitCtx.TaskID == nil {
			return fmt.Errorf("%s\n\nTip: use --task to specify the task ID explicitly", git.BranchNamingSuggestion(gitCtx.Branch))
		}
		taskID = gitCtx.TaskID.ID
	}

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
	var prDetail ghPRDetail
	if opts.prNumber > 0 {
		detail, err := fetchPRDetail(opts.prNumber, opts.repo)
		if err != nil {
			return err
		}
		prDetail = *detail
	} else {
		detail, err := fetchCurrentPRDetail()
		if err != nil {
			return err
		}
		prDetail = *detail
	}

	// Build the ClickUp info block for the PR body.
	clickupBlock := buildClickUpBlock(taskURL, taskName, taskStatus, priority, assigneeNames)

	// Update the PR body.
	newBody := upsertClickUpBlock(prDetail.Body, clickupBlock)
	if newBody != prDetail.Body {
		if err := updatePRBody(prDetail.Number, opts.repo, newBody); err != nil {
			return fmt.Errorf("failed to update PR body: %w", err)
		}
		fmt.Fprintf(ios.Out, "%s Updated PR #%d body with ClickUp task info\n",
			cs.Green("!"), prDetail.Number)
	} else {
		fmt.Fprintf(ios.Out, "PR #%d body already up to date\n", prDetail.Number)
	}

	// Post link comment on ClickUp task.
	repoSlug := opts.repo
	if repoSlug == "" {
		repoSlug = inferRepoFromURL(prDetail.URL)
	}

	linkText := fmt.Sprintf("%s#%d - %s", repoSlug, prDetail.Number, prDetail.Title)
	blocks := []commentBlock{
		{Text: "\xf0\x9f\x94\x97 "},
		{Text: "GitHub PR linked", Attributes: map[string]interface{}{"bold": true}},
		{Text: ": "},
		{Text: linkText, Attributes: map[string]interface{}{"link": prDetail.URL}},
		{Text: "\n"},
	}
	if err := postRichComment(opts.factory, taskID, blocks); err != nil {
		return err
	}
	fmt.Fprintf(ios.Out, "%s Linked PR #%d to task %s\n",
		cs.Green("!"), prDetail.Number, cs.Bold(taskID))

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
		// Replace existing block.
		return body[:startIdx] + block + body[endIdx+len(clickupBlockEnd):]
	}

	// Prepend the block.
	if body == "" {
		return block
	}
	return block + "\n\n" + body
}

func fetchPRDetail(number int, repo string) (*ghPRDetail, error) {
	args := []string{"pr", "view", strconv.Itoa(number), "--json", "number,title,body,url"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	cmd := exec.Command("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR #%d: %w", number, err)
	}
	var detail ghPRDetail
	if err := json.Unmarshal(out, &detail); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}
	return &detail, nil
}

func fetchCurrentPRDetail() (*ghPRDetail, error) {
	cmd := exec.Command("gh", "pr", "view", "--json", "number,title,body,url")
	out, err := cmd.Output()
	if err != nil {
		if isGHNotInstalled(err) {
			return nil, fmt.Errorf("the GitHub CLI (gh) is not installed or not in PATH\n\n" +
				"Install it from https://cli.github.com/ and authenticate with 'gh auth login'")
		}
		return nil, fmt.Errorf("failed to detect current PR: %w\n\n"+
			"Make sure you have an open PR for the current branch, or provide a PR number as an argument", err)
	}
	var detail ghPRDetail
	if err := json.Unmarshal(out, &detail); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}
	return &detail, nil
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
