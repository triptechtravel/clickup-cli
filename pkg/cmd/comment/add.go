package comment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type addOptions struct {
	factory *cmdutil.Factory
	taskID  string
	body    string
	editor  bool
}

// NewCmdAdd returns the "comment add" command.
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	opts := &addOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "add [TASK] [BODY]",
		Short: "Add a comment to a task",
		Long: `Add a comment to a ClickUp task.

If TASK is not provided, the task ID is auto-detected from the current git branch.
If BODY is not provided (or --editor is used), your editor opens for composing the comment.`,
		Example: `  # Add a comment to the task detected from the current branch
  clickup comment add "" "Fixed the login bug"

  # Add a comment to a specific task
  clickup comment add abc123 "Deployed to staging"

  # Open your editor to compose the comment
  clickup comment add --editor`,
		Args:              cobra.MaximumNArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 1 {
				opts.taskID = args[0]
			}
			if len(args) >= 2 {
				opts.body = args[1]
			}
			return addRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.editor, "editor", "e", false, "Open editor to compose comment body")

	return cmd
}

func addRun(opts *addOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID from git branch if not provided.
	taskID := opts.taskID
	if taskID == "" {
		gitCtx, err := opts.factory.GitContext()
		if err != nil {
			return fmt.Errorf("could not detect git context: %w\n\n%s", err,
				"Tip: provide the task ID as the first argument")
		}
		if gitCtx.TaskID == nil {
			return fmt.Errorf("%s", git.BranchNamingSuggestion(gitCtx.Branch))
		}
		taskID = gitCtx.TaskID.ID
		fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n", cs.Bold(taskID), cs.Cyan(gitCtx.Branch))
	}

	// Resolve comment body.
	body := opts.body
	if body == "" || opts.editor {
		p := prompter.New(ios)
		var err error
		body, err = p.Editor("Comment body", body, "*.md")
		if err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}
	}

	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	// Build and send the API request.
	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s/comment", taskID)
	payload, err := json.Marshal(map[string]string{
		"comment_text": body,
	})
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

	fmt.Fprintf(ios.Out, "%s Comment added to task %s\n", cs.Green("!"), cs.Bold(taskID))
	return nil
}
