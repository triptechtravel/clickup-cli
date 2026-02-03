package git

import (
	"os/exec"
	"strings"
)

// Client wraps git command execution.
type Client struct{}

// NewClient returns a new git client.
func NewClient() *Client {
	return &Client{}
}

// CurrentBranch returns the current git branch name.
func (c *Client) CurrentBranch() (string, error) {
	out, err := c.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// RemoteURL returns the URL of the specified remote.
func (c *Client) RemoteURL(remote string) (string, error) {
	out, err := c.run("remote", "get-url", remote)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// IsInsideWorkTree checks if the current directory is inside a git repository.
func (c *Client) IsInsideWorkTree() bool {
	_, err := c.run("rev-parse", "--is-inside-work-tree")
	return err == nil
}

// TopLevel returns the root directory of the git repository.
func (c *Client) TopLevel() (string, error) {
	out, err := c.run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// LatestCommitSHA returns the SHA of the latest commit.
func (c *Client) LatestCommitSHA() (string, error) {
	out, err := c.run("rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// LatestCommitMessage returns the subject line of the latest commit.
func (c *Client) LatestCommitMessage() (string, error) {
	out, err := c.run("log", "-1", "--format=%s")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (c *Client) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
