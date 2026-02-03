package git

import (
	"fmt"
	"regexp"
	"strings"
)

// RepoContext holds detected git context information.
type RepoContext struct {
	Branch    string
	RemoteURL string
	TaskID    *TaskIDResult
	RepoOwner string
	RepoName  string
}

var (
	// Matches GitHub SSH and HTTPS URLs
	githubSSHPattern   = regexp.MustCompile(`git@github\.com:([^/]+)/([^.]+)(?:\.git)?`)
	githubHTTPSPattern = regexp.MustCompile(`https://github\.com/([^/]+)/([^.]+?)(?:\.git)?$`)
)

// DetectContext gathers full git context from the current repository.
func DetectContext() (*RepoContext, error) {
	client := NewClient()

	if !client.IsInsideWorkTree() {
		return nil, fmt.Errorf("not inside a git repository")
	}

	ctx := &RepoContext{}

	branch, err := client.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to detect branch: %w", err)
	}
	ctx.Branch = branch
	ctx.TaskID = ExtractTaskID(branch)

	remoteURL, err := client.RemoteURL("origin")
	if err == nil {
		ctx.RemoteURL = remoteURL
		ctx.RepoOwner, ctx.RepoName = parseGitHubURL(remoteURL)
	}

	return ctx, nil
}

func parseGitHubURL(url string) (owner, repo string) {
	if matches := githubSSHPattern.FindStringSubmatch(url); len(matches) >= 3 {
		return matches[1], matches[2]
	}
	if matches := githubHTTPSPattern.FindStringSubmatch(url); len(matches) >= 3 {
		return matches[1], strings.TrimSuffix(matches[2], ".git")
	}
	return "", ""
}
