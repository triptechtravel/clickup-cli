// Package apiv3 provides typed wrappers for ClickUp API v3 operations.
//
// The do() helper handles auth, rate limiting, and error parsing via api.Client.
// All operation wrappers live in operations.gen.go (auto-generated).
package apiv3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/triptechtravel/clickup-cli/internal/api"
)

// do is a shared helper that sends a JSON request and decodes the response.
// path is relative to the v3 base (e.g. "workspaces/123/docs").
func do(ctx context.Context, client *api.Client, method, path string, body any, result any) error {
	rawURL := client.BaseURLV3() + "/" + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := api.HandleErrorResponse(resp); err != nil {
		return err
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
