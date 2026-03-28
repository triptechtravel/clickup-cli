package chat

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdChat returns the "chat" parent command.
func NewCmdChat(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Manage chat channels",
		Long:  "Send messages to ClickUp Chat channels.",
	}

	cmd.AddCommand(NewCmdChatSend(f))

	return cmd
}
