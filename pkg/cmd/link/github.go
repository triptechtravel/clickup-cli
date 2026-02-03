package link

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ghPR holds the JSON output from `gh pr view`.
type ghPR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	URL    string `json:"url"`
}

// fetchPR fetches a specific PR by number using the GitHub CLI.
func fetchPR(number int, repo string) (*ghPR, error) {
	args := []string{"pr", "view", strconv.Itoa(number), "--json", "number,title,body,url"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	return runGHPR(args, fmt.Sprintf("failed to fetch PR #%d", number))
}

// fetchCurrentPR fetches the PR for the current branch using the GitHub CLI.
func fetchCurrentPR() (*ghPR, error) {
	args := []string{"pr", "view", "--json", "number,title,body,url"}
	return runGHPR(args, "failed to detect current PR.\n\n"+
		"Make sure you have an open PR for the current branch, or provide a PR number as an argument")
}

// runGHPR executes a gh pr command and parses the JSON output.
func runGHPR(args []string, errContext string) (*ghPR, error) {
	cmd := exec.Command("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		if isGHNotInstalled(err) {
			return nil, ghNotInstalledError()
		}
		return nil, fmt.Errorf("%s: %w", errContext, err)
	}
	var pr ghPR
	if err := json.Unmarshal(out, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}
	return &pr, nil
}

// buildPREntry creates a linkEntry for a pull request.
func buildPREntry(repoSlug string, number int, title, url string) linkEntry {
	return linkEntry{
		Prefix: fmt.Sprintf("%s#%d", repoSlug, number),
		Line:   fmt.Sprintf("[%s#%d — %s](%s)", repoSlug, number, title, url),
	}
}

// inferRepoFromURL extracts "owner/repo" from a GitHub PR URL.
func inferRepoFromURL(prURL string) string {
	prURL = strings.TrimPrefix(prURL, "https://github.com/")
	parts := strings.Split(prURL, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return ""
}

// isGHNotInstalled checks if the error indicates the gh CLI is not found.
func isGHNotInstalled(err error) bool {
	_, ok := err.(*exec.Error)
	return ok
}

// ghNotInstalledError returns a user-friendly error for missing gh CLI.
func ghNotInstalledError() error {
	return fmt.Errorf("the GitHub CLI (gh) is not installed or not in PATH\n\n" +
		"Install it from https://cli.github.com/ and authenticate with 'gh auth login'")
}
