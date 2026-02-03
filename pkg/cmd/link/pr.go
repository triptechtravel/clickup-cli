package link

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type prOptions struct {
	factory  *cmdutil.Factory
	prNumber int
	taskID   string
	repo     string
}

// ghPRInfo holds the JSON output from `gh pr view`.
type ghPRInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// NewCmdLinkPR returns the "link pr" command.
func NewCmdLinkPR(f *cmdutil.Factory) *cobra.Command {
	opts := &prOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "pr [NUMBER]",
		Short: "Link a GitHub PR to a ClickUp task",
		Long: `Link a GitHub pull request to a ClickUp task by posting a comment.

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
	var prInfo ghPRInfo
	if opts.prNumber > 0 {
		info, err := fetchPRByNumber(opts.prNumber, opts.repo)
		if err != nil {
			return err
		}
		prInfo = *info
	} else {
		info, err := fetchCurrentPR()
		if err != nil {
			return err
		}
		prInfo = *info
	}

	// Infer repo slug from PR URL if we don't have it yet.
	if repoSlug == "" {
		repoSlug = inferRepoFromURL(prInfo.URL)
	}

	// Build rich text comment blocks.
	linkText := fmt.Sprintf("%s#%d - %s", repoSlug, prInfo.Number, prInfo.Title)
	blocks := []commentBlock{
		{Text: "\xf0\x9f\x94\x97 "},
		{Text: "GitHub PR linked", Attributes: map[string]interface{}{"bold": true}},
		{Text: ": "},
		{Text: linkText, Attributes: map[string]interface{}{"link": prInfo.URL}},
		{Text: "\n"},
	}

	// Post the comment.
	if err := postRichComment(opts.factory, taskID, blocks); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "%s Linked PR #%d to task %s\n",
		cs.Green("!"), prInfo.Number, cs.Bold(taskID))
	return nil
}

// fetchCurrentPR uses `gh pr view` to get the PR for the current branch.
func fetchCurrentPR() (*ghPRInfo, error) {
	cmd := exec.Command("gh", "pr", "view", "--json", "number,title,url")
	out, err := cmd.Output()
	if err != nil {
		if isGHNotInstalled(err) {
			return nil, fmt.Errorf("the GitHub CLI (gh) is not installed or not in PATH\n\n" +
				"Install it from https://cli.github.com/ and authenticate with 'gh auth login'")
		}
		return nil, fmt.Errorf("failed to detect current PR: %w\n\n"+
			"Make sure you have an open PR for the current branch, or provide a PR number as an argument", err)
	}

	var info ghPRInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}
	return &info, nil
}

// fetchPRByNumber uses `gh pr view NUMBER` to get a specific PR.
func fetchPRByNumber(number int, repo string) (*ghPRInfo, error) {
	args := []string{"pr", "view", strconv.Itoa(number), "--json", "number,title,url"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	cmd := exec.Command("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		if isGHNotInstalled(err) {
			return nil, fmt.Errorf("the GitHub CLI (gh) is not installed or not in PATH\n\n" +
				"Install it from https://cli.github.com/ and authenticate with 'gh auth login'")
		}
		return nil, fmt.Errorf("failed to fetch PR #%d: %w", number, err)
	}

	var info ghPRInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}
	return &info, nil
}

// inferRepoFromURL extracts "owner/repo" from a GitHub PR URL.
func inferRepoFromURL(prURL string) string {
	// URL format: https://github.com/owner/repo/pull/123
	prURL = strings.TrimPrefix(prURL, "https://github.com/")
	parts := strings.Split(prURL, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return ""
}

// isGHNotInstalled checks if the error indicates the gh CLI is not found.
func isGHNotInstalled(err error) bool {
	var execErr *exec.Error
	if ok := isExecError(err, &execErr); ok {
		return true
	}
	return false
}

func isExecError(err error, target **exec.Error) bool {
	switch e := err.(type) {
	case *exec.Error:
		*target = e
		return true
	default:
		return false
	}
}

// commentBlock represents a rich text block in a ClickUp comment.
type commentBlock struct {
	Text       string                 `json:"text"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// richCommentPayload is the request body for posting a rich text comment.
type richCommentPayload struct {
	Comment []commentBlock `json:"comment"`
}

// postRichComment posts a rich text comment to a ClickUp task.
func postRichComment(f *cmdutil.Factory, taskID string, blocks []commentBlock) error {
	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s/comment", taskID)
	payload, err := json.Marshal(richCommentPayload{Comment: blocks})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.DoRequest(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
