package apiv3

import (
	"context"
	"fmt"

	"github.com/triptechtravel/clickup-cli/api/clickupv3"
	"github.com/triptechtravel/clickup-cli/internal/api"
)

// ListAttachments fetches attachments for a task (or other entity) via the v3 API.
// entityType is "attachments" for task attachments, "custom_fields" for file custom fields.
func ListAttachments(ctx context.Context, client *api.Client, workspaceID, entityType, entityID string, opts ...GetParentEntityAttachmentsParams) (*clickupv3.AttachmentsPublicAPIAttachmentsControllerGetParentEntityAttachments200Response, error) {
	var resp clickupv3.AttachmentsPublicAPIAttachmentsControllerGetParentEntityAttachments200Response
	path := fmt.Sprintf("workspaces/%s/%s/%s/attachments", workspaceID, entityType, entityID)
	if len(opts) > 0 {
		path += opts[0].encode()
	}
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
