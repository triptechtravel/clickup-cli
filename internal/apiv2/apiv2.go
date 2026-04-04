// Package apiv2 provides typed wrappers for ClickUp API v2 operations that
// are missing from the go-clickup library.
//
// go-clickup handles most V2 operations correctly. This package fills the
// gaps documented in api/GO_CLICKUP_GAPS.md:
//   - UpdateTask with points and markdown_content
//   - Time entry CRUD
//   - Comment replies (threaded comments)
//   - Space tags
//
// All wrappers use the existing api.Client for HTTP transport (auth, rate
// limiting) and auto-generated clickupv2 types for request/response
// serialization.
package apiv2

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
func do(ctx context.Context, client *api.Client, method, path string, body any, result any) error {
	url := client.BaseURL() + "/" + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
