package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type sendOptions struct {
	factory   *cmdutil.Factory
	channelID string
	body      string
	notifyAll bool
}

// NewCmdChatSend returns the "chat send" command.
func NewCmdChatSend(f *cmdutil.Factory) *cobra.Command {
	opts := &sendOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "send <channel-id> <message>",
		Short: "Send a message to a ClickUp Chat channel",
		Long: `Post a message to a ClickUp Chat channel (view).

The channel ID can be found in the Chat URL:
  https://app.clickup.com/<workspace>/chat/r/<channel-id>

For example, if the URL is:
  https://app.clickup.com/20503057/chat/r/khpgh-10335
then the channel ID is "khpgh-10335".`,
		Example: `  # Send a message to a chat channel
  clickup chat send khpgh-10335 "Deploy completed successfully"

  # Send with notifications
  clickup chat send khpgh-10335 "Urgent: server down" --notify`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channelID = args[0]
			opts.body = args[1]
			return sendRun(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.notifyAll, "notify", false, "Notify all channel members")

	return cmd
}

func sendRun(opts *sendOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.clickup.com/api/v2/view/%s/comment", opts.channelID)

	payload, err := json.Marshal(map[string]interface{}{
		"comment_text": opts.body,
		"notify_all":   opts.notifyAll,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
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

	fmt.Fprintf(ios.Out, "%s Message sent to channel %s\n", cs.Green("!"), cs.Bold(opts.channelID))

	return nil
}
