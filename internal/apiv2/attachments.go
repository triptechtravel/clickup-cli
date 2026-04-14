package apiv2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/triptechtravel/clickup-cli/internal/api"
)

// AttachmentResponse is the response from CreateTaskAttachment.
type AttachmentResponse struct {
	ID             string `json:"id"`
	Version        string `json:"version"`
	Date           int    `json:"date"`
	Title          string `json:"title"`
	Extension      string `json:"extension"`
	ThumbnailSmall string `json:"thumbnail_small"`
	ThumbnailLarge string `json:"thumbnail_large"`
	URL            string `json:"url"`
}

// CreateTaskAttachmentParams holds optional parameters for attachment upload.
type CreateTaskAttachmentParams struct {
	CustomTaskIDs bool
	TeamID        string
}

// CreateTaskAttachment uploads a file as an attachment to a task.
func CreateTaskAttachment(ctx context.Context, client *api.Client, taskID, filename string, reader io.Reader, params ...CreateTaskAttachmentParams) (*AttachmentResponse, error) {
	path := fmt.Sprintf("task/%s/attachment", taskID)
	if len(params) > 0 {
		p := params[0]
		sep := "?"
		if p.CustomTaskIDs {
			path += sep + "custom_task_ids=true"
			sep = "&"
		}
		if p.TeamID != "" {
			path += sep + "team_id=" + p.TeamID
		}
	}

	contents, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read attachment: %w", err)
	}

	var buf bytes.Buffer
	multipartWriter := multipart.NewWriter(&buf)
	part, err := multipartWriter.CreateFormFile("attachment", filepath.Base(filename))
	if err != nil {
		return nil, fmt.Errorf("create multipart field: %w", err)
	}
	if _, err := part.Write(contents); err != nil {
		return nil, fmt.Errorf("write attachment data: %w", err)
	}
	if err := multipartWriter.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	url := client.BaseURL() + "/" + path
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	resp, err := client.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result AttachmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
