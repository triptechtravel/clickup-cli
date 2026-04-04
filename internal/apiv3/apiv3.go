// Package apiv3 provides typed wrappers for ClickUp API v3 operations.
//
// All wrappers use the existing api.Client for HTTP transport (auth, rate
// limiting) and route requests to the v3 API base URL.
package apiv3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

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

// DocParent represents a parent location (space, folder, list, etc.) for a Doc.
type DocParent struct {
	ID   string  `json:"id"`
	Type float32 `json:"type"`
}

// DocCoreResult holds common fields returned by doc list/get endpoints.
type DocCoreResult struct {
	ID          string    `json:"id"`
	Name        string   `json:"name"`
	Deleted     bool     `json:"deleted"`
	Archived    bool     `json:"archived"`
	Visibility  string   `json:"visibility"`
	DateCreated string   `json:"date_created"`
	DateUpdated string   `json:"date_updated"`
	Creator     DocUser  `json:"creator"`
	Parent      DocParent `json:"parent"`
	Workspace   struct {
		ID string `json:"id"`
	} `json:"workspace"`
}

// DocUser holds creator information returned by doc endpoints.
type DocUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// DocsSearchResult is returned by SearchDocs.
type DocsSearchResult struct {
	Docs       []DocCoreResult `json:"docs"`
	NextCursor string          `json:"next_cursor"`
}

// SearchDocsParams holds optional query parameters for SearchDocs.
type SearchDocsParams struct {
	Deleted    bool
	Archived   bool
	Creator    int
	ParentID   string
	ParentType int
	Limit      int
	Cursor     string
}

// SearchDocs fetches a page of Docs in the given workspace.
func SearchDocs(ctx context.Context, client *api.Client, workspaceID string, params SearchDocsParams) (*DocsSearchResult, error) {
	q := url.Values{}
	if params.Deleted {
		q.Set("deleted", "true")
	}
	if params.Archived {
		q.Set("archived", "true")
	}
	if params.Creator != 0 {
		q.Set("creator", fmt.Sprintf("%d", params.Creator))
	}
	if params.ParentID != "" {
		q.Set("parent_id", params.ParentID)
		if params.ParentType != 0 {
			q.Set("parent_type", fmt.Sprintf("%d", params.ParentType))
		}
	}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	if params.Cursor != "" {
		q.Set("cursor", params.Cursor)
	}

	path := "workspaces/" + url.PathEscape(workspaceID) + "/docs"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	var result DocsSearchResult
	if err := do(ctx, client, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDoc fetches a single Doc by ID.
func GetDoc(ctx context.Context, client *api.Client, workspaceID, docID string) (*DocCoreResult, error) {
	path := fmt.Sprintf("workspaces/%s/docs/%s",
		url.PathEscape(workspaceID),
		url.PathEscape(docID),
	)
	var result DocCoreResult
	if err := do(ctx, client, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateDocRequest holds the body for creating a Doc.
type CreateDocRequest struct {
	Name       string     `json:"name"`
	CreatePage bool       `json:"create_page"`
	Parent     *DocParent `json:"parent,omitempty"`
	Visibility string     `json:"visibility,omitempty"`
}

// CreateDoc creates a new Doc in the given workspace.
func CreateDoc(ctx context.Context, client *api.Client, workspaceID string, req *CreateDocRequest) (*DocCoreResult, error) {
	path := fmt.Sprintf("workspaces/%s/docs", url.PathEscape(workspaceID))
	var result DocCoreResult
	if err := do(ctx, client, "POST", path, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PageRef is a summary entry in the page listing response.
type PageRef struct {
	ID           string     `json:"id"`
	DocID        string     `json:"doc_id"`
	Name         string     `json:"name"`
	SubTitle     string     `json:"sub_title"`
	OrderIndex   int        `json:"order_index"`
	Pages        []PageRef  `json:"pages"`
}

// PagesListResult is returned by GetDocPages.
type PagesListResult struct {
	Pages []PageRef `json:"pages"`
}

// GetDocPages fetches the pages for a Doc (tree structure).
func GetDocPages(ctx context.Context, client *api.Client, workspaceID, docID string, maxDepth int) (*PagesListResult, error) {
	path := fmt.Sprintf("workspaces/%s/docs/%s/pages",
		url.PathEscape(workspaceID),
		url.PathEscape(docID),
	)
	if maxDepth >= 0 {
		q := url.Values{}
		q.Set("max_page_depth", fmt.Sprintf("%d", maxDepth))
		path += "?" + q.Encode()
	}
	var result PagesListResult
	if err := do(ctx, client, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PageDetail is the full page returned by get-page and create-page endpoints.
type PageDetail struct {
	ID            string    `json:"id"`
	DocID         string    `json:"doc_id"`
	Name          string    `json:"name"`
	SubTitle      string    `json:"sub_title"`
	Content       string    `json:"content"`
	ContentFormat string    `json:"content_format"`
	OrderIndex    int       `json:"order_index"`
	DateCreated   string    `json:"date_created"`
	DateUpdated   string    `json:"date_updated"`
	Creator       DocUser   `json:"creator"`
	Pages         []PageRef `json:"pages"`
}

// CreatePageRequest holds the body for creating a page.
type CreatePageRequest struct {
	Name          string `json:"name"`
	ParentPageID  string `json:"parent_page_id,omitempty"`
	SubTitle      string `json:"sub_title,omitempty"`
	Content       string `json:"content,omitempty"`
	ContentFormat string `json:"content_format,omitempty"`
}

// CreatePage creates a new page within a Doc.
func CreatePage(ctx context.Context, client *api.Client, workspaceID, docID string, req *CreatePageRequest) (*PageDetail, error) {
	path := fmt.Sprintf("workspaces/%s/docs/%s/pages",
		url.PathEscape(workspaceID),
		url.PathEscape(docID),
	)
	var result PageDetail
	if err := do(ctx, client, "POST", path, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// EditPageRequest holds the body for editing a page.
type EditPageRequest struct {
	Name            string `json:"name,omitempty"`
	SubTitle        string `json:"sub_title,omitempty"`
	Content         string `json:"content,omitempty"`
	ContentFormat   string `json:"content_format,omitempty"`
	ContentEditMode string `json:"content_edit_mode,omitempty"`
}

// EditPage updates an existing page.
func EditPage(ctx context.Context, client *api.Client, workspaceID, docID, pageID string, req *EditPageRequest) (*PageDetail, error) {
	path := fmt.Sprintf("workspaces/%s/docs/%s/pages/%s",
		url.PathEscape(workspaceID),
		url.PathEscape(docID),
		url.PathEscape(pageID),
	)
	var result PageDetail
	if err := do(ctx, client, "PUT", path, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPageParams holds optional query parameters for GetPage.
type GetPageParams struct {
	ContentFormat string
}

// GetPage fetches a single page from a Doc.
func GetPage(ctx context.Context, client *api.Client, workspaceID, docID, pageID string, params GetPageParams) (*PageDetail, error) {
	path := fmt.Sprintf("workspaces/%s/docs/%s/pages/%s",
		url.PathEscape(workspaceID),
		url.PathEscape(docID),
		url.PathEscape(pageID),
	)
	if params.ContentFormat != "" {
		q := url.Values{}
		q.Set("content_format", params.ContentFormat)
		path += "?" + q.Encode()
	}
	var result PageDetail
	if err := do(ctx, client, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
