package apiv3_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
)

// testV3Server creates a test server and a client configured to talk to it
// at the /api/v3/ base path.
func testV3Server(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, api.NewTestClient(server.URL)
}

func TestSearchDocs_ReturnsResults(t *testing.T) {
	const responseBody = `{"docs":[{"id":"doc1","name":"Runbook","visibility":"PUBLIC","deleted":false,"archived":false}],"next_cursor":""}`

	server, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v3/workspaces/ws1/docs")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(responseBody))
	})
	_ = server

	result, err := apiv3.SearchDocs(context.Background(), client, "ws1", apiv3.SearchDocsParams{})
	require.NoError(t, err)
	require.Len(t, result.Docs, 1)
	assert.Equal(t, "doc1", result.Docs[0].ID)
	assert.Equal(t, "Runbook", result.Docs[0].Name)
}

func TestSearchDocs_BuildsQueryParams(t *testing.T) {
	var capturedURL string

	server, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"docs":[],"next_cursor":""}`))
	})
	_ = server

	_, err := apiv3.SearchDocs(context.Background(), client, "ws1", apiv3.SearchDocsParams{
		Deleted:    true,
		Archived:   true,
		Creator:    42,
		ParentID:   "parent/id",
		ParentType: 4,
		Limit:      10,
		Cursor:     "tok",
	})
	require.NoError(t, err)

	assert.Contains(t, capturedURL, "deleted=true")
	assert.Contains(t, capturedURL, "archived=true")
	assert.Contains(t, capturedURL, "creator=42")
	assert.Contains(t, capturedURL, "parent_id=parent%2Fid")
	assert.Contains(t, capturedURL, "parent_type=4")
	assert.Contains(t, capturedURL, "limit=10")
	assert.Contains(t, capturedURL, "cursor=tok")
}

func TestGetDoc_ReturnsDoc(t *testing.T) {
	const responseBody = `{"id":"doc1","name":"Runbook","visibility":"PUBLIC","deleted":false,"archived":false,"date_created":"1714000000000","date_updated":"1714100000000"}`

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v3/workspaces/ws1/docs/doc1")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(responseBody))
	})

	doc, err := apiv3.GetDoc(context.Background(), client, "ws1", "doc1")
	require.NoError(t, err)
	assert.Equal(t, "doc1", doc.ID)
	assert.Equal(t, "Runbook", doc.Name)
	assert.Equal(t, "PUBLIC", doc.Visibility)
}

func TestCreateDoc_SendsBody(t *testing.T) {
	var capturedBody map[string]interface{}

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id":"newdoc","name":"My Doc","visibility":"PUBLIC"}`))
	})

	req := &apiv3.CreateDocRequest{
		Name:       "My Doc",
		CreatePage: true,
		Visibility: "PUBLIC",
	}
	doc, err := apiv3.CreateDoc(context.Background(), client, "ws1", req)
	require.NoError(t, err)
	assert.Equal(t, "newdoc", doc.ID)
	assert.Equal(t, "My Doc", capturedBody["name"])
	assert.Equal(t, true, capturedBody["create_page"])
	assert.Equal(t, "PUBLIC", capturedBody["visibility"])
}

func TestCreateDoc_WithParent(t *testing.T) {
	var capturedBody map[string]interface{}

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id":"newdoc","name":"My Doc"}`))
	})

	req := &apiv3.CreateDocRequest{
		Name: "My Doc",
		Parent: &apiv3.DocParent{
			ID:   "space123",
			Type: 4,
		},
	}
	_, err := apiv3.CreateDoc(context.Background(), client, "ws1", req)
	require.NoError(t, err)

	parent, ok := capturedBody["parent"].(map[string]interface{})
	require.True(t, ok, "expected parent object in body")
	assert.Equal(t, "space123", parent["id"])
}

func TestGetDocPages_ReturnsPagesTree(t *testing.T) {
	const responseBody = `{"pages":[{"id":"p1","name":"Intro","doc_id":"doc1","pages":[]},{"id":"p2","name":"Setup","doc_id":"doc1","pages":[{"id":"p3","name":"Advanced","doc_id":"doc1"}]}]}`

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v3/workspaces/ws1/docs/doc1/pages")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(responseBody))
	})

	result, err := apiv3.GetDocPages(context.Background(), client, "ws1", "doc1", -1)
	require.NoError(t, err)
	require.Len(t, result.Pages, 2)
	assert.Equal(t, "p1", result.Pages[0].ID)
	require.Len(t, result.Pages[1].Pages, 1)
	assert.Equal(t, "p3", result.Pages[1].Pages[0].ID)
}

func TestGetDocPages_PassesMaxDepth(t *testing.T) {
	var capturedQuery string

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"pages":[]}`))
	})

	_, err := apiv3.GetDocPages(context.Background(), client, "ws1", "doc1", 2)
	require.NoError(t, err)
	assert.Contains(t, capturedQuery, "max_page_depth=2")
}

func TestCreatePage_SendsBody(t *testing.T) {
	var capturedBody map[string]interface{}

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id":"page1","name":"Intro","doc_id":"doc1"}`))
	})

	req := &apiv3.CreatePageRequest{
		Name:          "Intro",
		Content:       "Hello world",
		ContentFormat: "text/md",
	}
	page, err := apiv3.CreatePage(context.Background(), client, "ws1", "doc1", req)
	require.NoError(t, err)
	assert.Equal(t, "page1", page.ID)
	assert.Equal(t, "Intro", capturedBody["name"])
	assert.Equal(t, "Hello world", capturedBody["content"])
	assert.Equal(t, "text/md", capturedBody["content_format"])
}

func TestEditPage_SendsBody(t *testing.T) {
	var capturedBody map[string]interface{}

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v3/workspaces/ws1/docs/doc1/pages/page1")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id":"page1","name":"Updated","doc_id":"doc1"}`))
	})

	req := &apiv3.EditPageRequest{
		Content:         "New content",
		ContentEditMode: "append",
	}
	page, err := apiv3.EditPage(context.Background(), client, "ws1", "doc1", "page1", req)
	require.NoError(t, err)
	assert.Equal(t, "page1", page.ID)
	assert.Equal(t, "New content", capturedBody["content"])
	assert.Equal(t, "append", capturedBody["content_edit_mode"])
}

func TestGetPage_PassesContentFormat(t *testing.T) {
	var capturedQuery string

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id":"page1","name":"Intro","doc_id":"doc1"}`))
	})

	_, err := apiv3.GetPage(context.Background(), client, "ws1", "doc1", "page1", apiv3.GetPageParams{
		ContentFormat: "text/md",
	})
	require.NoError(t, err)
	assert.Contains(t, capturedQuery, "content_format=text%2Fmd")
}

func TestGetDoc_APIError(t *testing.T) {
	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"err":"Doc not found","ECODE":"DOC_000"}`))
	})

	_, err := apiv3.GetDoc(context.Background(), client, "ws1", "notexist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestPathEscaping_SpecialChars(t *testing.T) {
	var capturedRawPath string

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		// r.URL.RawPath preserves the percent-encoding sent by the client.
		// r.URL.Path decodes it (unescaped), so we check RawPath when non-empty.
		if r.URL.RawPath != "" {
			capturedRawPath = r.URL.RawPath
		} else {
			capturedRawPath = r.URL.Path
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id":"doc/1","name":"Test","visibility":"PUBLIC"}`))
	})

	_, err := apiv3.GetDoc(context.Background(), client, "ws1", "doc/1")
	require.NoError(t, err)
	// The slash in "doc/1" should be path-escaped in the raw path.
	assert.Contains(t, capturedRawPath, "doc%2F1")
}
