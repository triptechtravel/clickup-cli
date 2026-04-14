package apiv3

// Typed wrappers for chat operations whose generated signatures lack response
// decoding (the codegen emits `nil` as the response target for these endpoints).
// These functions shadow the generated ones by using identical names — the Go
// linker will reject duplicate symbols, so the generated file must NOT define
// these functions.  When the codegen is re-run it will recreate the untyped
// versions; regenerate this file or update the codegen template to fix.

import (
	"context"
	"fmt"

	clickupv3 "github.com/triptechtravel/clickup-cli/api/clickupv3"
	"github.com/triptechtravel/clickup-cli/internal/api"
)

// ListChatChannels retrieves Chat channels for a workspace with typed response.
func ListChatChannels(ctx context.Context, client *api.Client, workspaceId string, opts ...GetChatChannelsParams) (*clickupv3.ChatPublicAPIChatChannelsControllerGetChatChannels200Response, error) {
	var resp clickupv3.ChatPublicAPIChatChannelsControllerGetChatChannels200Response
	path := fmt.Sprintf("workspaces/%s/chat/channels", workspaceId)
	if len(opts) > 0 {
		path += opts[0].encode()
	}
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListChatMessages retrieves messages for a Chat channel with typed response.
func ListChatMessages(ctx context.Context, client *api.Client, workspaceId string, channelId string, opts ...GetChatMessagesParams) (*clickupv3.CommentPublicAPIChatMessagesControllerGetChatMessages200Response, error) {
	var resp clickupv3.CommentPublicAPIChatMessagesControllerGetChatMessages200Response
	path := fmt.Sprintf("workspaces/%s/chat/channels/%s/messages", workspaceId, channelId)
	if len(opts) > 0 {
		path += opts[0].encode()
	}
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
