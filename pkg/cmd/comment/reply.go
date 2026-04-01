package comment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type replyOptions struct {
	factory   *cmdutil.Factory
	commentID string
	body      string
	editor    bool
}

// NewCmdReply returns the "comment reply" command.
func NewCmdReply(f *cmdutil.Factory) *cobra.Command {
	opts := &replyOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "reply <comment-id> [BODY]",
		Short: "Reply to a comment thread",
		Long: `Reply to an existing comment on a ClickUp task, creating a threaded reply.

Use 'clickup comment list <task> --json' to find comment IDs.
Use @username in the body to @mention workspace members.`,
		Example: `  # Reply to a specific comment
  clickup comment reply 90160175975219 "Yes, that's confirmed"

  # Reply with @mentions
  clickup comment reply 90160175975219 "@Michelle confirmed, BookEasy only"

  # Open editor to compose the reply
  clickup comment reply 90160175975219 --editor`,
		Args:              cobra.RangeArgs(1, 2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.commentID = args[0]
			if len(args) >= 2 {
				opts.body = args[1]
			}
			return replyRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.editor, "editor", "e", false, "Open editor to compose reply body")

	return cmd
}

func replyRun(opts *replyOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve reply body.
	body := opts.body
	if body == "" || opts.editor {
		p := prompter.New(ios)
		var err error
		body, err = p.Editor("Reply body", body, "*.md")
		if err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}
	}

	if body == "" {
		return fmt.Errorf("reply body cannot be empty")
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.clickup.com/api/v2/comment/%s/reply", opts.commentID)

	// Build reply payload, resolving @mentions to real ClickUp user tags.
	var payload []byte
	if strings.Contains(body, "@") {
		if members, mErr := fetchWorkspaceMembers(opts.factory, client); mErr == nil && len(members) > 0 {
			if blocks, resolved := buildCommentBlocks(body, members); len(resolved) > 0 {
				for _, name := range resolved {
					fmt.Fprintf(ios.ErrOut, "Mentioning %s\n", cs.Bold("@"+name))
				}
				payload, err = json.Marshal(map[string]interface{}{"comment": blocks})
				if err != nil {
					return fmt.Errorf("failed to marshal reply: %w", err)
				}
			}
		}
	}
	if payload == nil {
		payload, err = json.Marshal(map[string]string{"comment_text": body})
		if err != nil {
			return fmt.Errorf("failed to marshal reply: %w", err)
		}
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

	fmt.Fprintf(ios.Out, "%s Reply added to comment %s\n", cs.Green("!"), cs.Bold(opts.commentID))

	return nil
}
