package comment

import (
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type deleteOptions struct {
	factory   *cmdutil.Factory
	commentID string
	confirm   bool
}

// NewCmdDelete returns the "comment delete" command.
func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "delete <COMMENT_ID>",
		Short: "Delete a comment",
		Long: `Delete a comment from a ClickUp task.

COMMENT_ID is required. Find comment IDs with 'clickup comment list TASK_ID --json'.
Use --yes to skip the confirmation prompt.`,
		Example: `  # Delete a comment (with confirmation)
  clickup comment delete 90160162431205

  # Delete without confirmation
  clickup comment delete 90160162431205 --yes

  # Find comment IDs first
  clickup comment list 86d1rn980 --json | jq '.[].id'`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.commentID = args[0]
			return deleteRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func deleteRun(opts *deleteOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	if !opts.confirm && ios.IsTerminal() {
		p := prompter.New(ios)
		ok, err := p.Confirm(fmt.Sprintf("Delete comment %s?", opts.commentID), false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(ios.ErrOut, "Cancelled.")
			return nil
		}
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.clickup.com/api/v2/comment/%s", opts.commentID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	fmt.Fprintf(ios.Out, "%s Comment %s deleted\n", cs.Green("!"), cs.Bold(opts.commentID))
	return nil
}
