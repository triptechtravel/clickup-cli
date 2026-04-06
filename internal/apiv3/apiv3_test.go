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
	clickupv3 "github.com/triptechtravel/clickup-cli/api/clickupv3"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
)

// testV3Server creates a test server and a client configured to talk to it.
// The client's BaseURLV3() will point to /api/v3/ on the test server.
func testV3Server(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, api.NewTestClient(server.URL)
}

func TestSearchDocsPublic_ReturnsResults(t *testing.T) {
	const responseBody = `{"docs":[{"id":"doc1","name":"Runbook","public":true,"deleted":false,"archived":false,"date_created":0,"creator":0,"workspace_id":0,"parent":{"id":"","type":0},"type":0}],"next_cursor":null}`

	server, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v3/workspaces/ws1/docs")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(responseBody))
	})
	_ = server

	result, err := apiv3.SearchDocsPublic(context.Background(), client, "ws1")
	require.NoError(t, err)
	require.Len(t, result.Docs, 1)
	assert.Equal(t, "doc1", result.Docs[0].ID)
	assert.Equal(t, "Runbook", result.Docs[0].Name)
}

func TestSearchDocsPublic_BuildsQueryParams(t *testing.T) {
	var capturedURL string

	server, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"docs":[],"next_cursor":null}`))
	})
	_ = server

	_, err := apiv3.SearchDocsPublic(context.Background(), client, "ws1",
		apiv3.SearchDocsPublicParams{
			Deleted:    true,
			Archived:   true,
			Creator:    42,
			ParentId:   "parent/id",
			ParentType: "SPACE",
			Limit:      10,
			Cursor:     "tok",
		},
	)
	require.NoError(t, err)

	assert.Contains(t, capturedURL, "deleted=true")
	assert.Contains(t, capturedURL, "archived=true")
	assert.Contains(t, capturedURL, "creator=42")
	assert.Contains(t, capturedURL, "parent_id=parent%2Fid")
	assert.Contains(t, capturedURL, "parent_type=SPACE")
	assert.Contains(t, capturedURL, "limit=10")
	assert.Contains(t, capturedURL, "cursor=tok")
}

func TestGetDocPublic_ReturnsDoc(t *testing.T) {
	const responseBody = `{"id":"doc1","name":"Runbook","public":true,"deleted":false,"archived":false,"date_created":1714000000000,"date_updated":1714100000000,"creator":0,"workspace_id":0,"parent":{"id":"","type":0},"type":0}`

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v3/workspaces/ws1/docs/doc1")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(responseBody))
	})

	doc, err := apiv3.GetDocPublic(context.Background(), client, "ws1", "doc1")
	require.NoError(t, err)
	assert.Equal(t, "doc1", doc.ID)
	assert.Equal(t, "Runbook", doc.Name)
	assert.True(t, doc.Public)
}

func TestCreateDocPublic_SendsBody(t *testing.T) {
	var capturedBody map[string]interface{}

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id":"newdoc","name":"My Doc","public":true,"deleted":false,"archived":false,"date_created":0,"creator":0,"workspace_id":0,"parent":{"id":"","type":0},"type":0}`))
	})

	name := "My Doc"
	createPage := true
	req := &clickupv3.PublicDocsCreateDocOptionsDto{
		Name:       &name,
		CreatePage: &createPage,
	}
	doc, err := apiv3.CreateDocPublic(context.Background(), client, "ws1", req)
	require.NoError(t, err)
	assert.Equal(t, "newdoc", doc.ID)
	assert.Equal(t, "My Doc", capturedBody["name"])
	assert.Equal(t, true, capturedBody["create_page"])
}

func TestCreateDocPublic_WithParent(t *testing.T) {
	var capturedBody map[string]interface{}

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id":"newdoc","name":"My Doc","public":false,"deleted":false,"archived":false,"date_created":0,"creator":0,"workspace_id":0,"parent":{"id":"space123","type":4},"type":0}`))
	})

	name := "My Doc"
	req := &clickupv3.PublicDocsCreateDocOptionsDto{
		Name: &name,
		Parent: &clickupv3.PublicDocsCreateDocOptionsDtoParent{
			ID:   "space123",
			Type: 4,
		},
	}
	_, err := apiv3.CreateDocPublic(context.Background(), client, "ws1", req)
	require.NoError(t, err)

	parent, ok := capturedBody["parent"].(map[string]interface{})
	require.True(t, ok, "expected parent object in body")
	assert.Equal(t, "space123", parent["id"])
}

func TestGetDocPagesPublic_ReturnsPagesTree(t *testing.T) {
	const responseBody = `[{"id":"p1","name":"Intro","doc_id":"doc1","workspace_id":0,"authors":[]},{"id":"p2","name":"Setup","doc_id":"doc1","workspace_id":0,"authors":[],"pages":[{"id":"p3","name":"Advanced","doc_id":"doc1","workspace_id":0,"authors":[]}]}]`

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v3/workspaces/ws1/docs/doc1/pages")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(responseBody))
	})

	result, err := apiv3.GetDocPagesPublic(context.Background(), client, "ws1", "doc1")
	require.NoError(t, err)
	require.Len(t, *result, 2)
	assert.Equal(t, "p1", (*result)[0].ID)
	require.NotNil(t, (*result)[1].Pages)
	require.Len(t, (*result)[1].Pages, 1)
	assert.Equal(t, "p3", ((*result)[1].Pages)[0].ID)
}

func TestGetDocPagesPublic_PassesMaxDepth(t *testing.T) {
	var capturedQuery string

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`[]`))
	})

	_, err := apiv3.GetDocPagesPublic(context.Background(), client, "ws1", "doc1",
		apiv3.GetDocPagesPublicParams{MaxPageDepth: 2},
	)
	require.NoError(t, err)
	assert.Contains(t, capturedQuery, "max_page_depth=2")
}

func TestCreatePagePublic_SendsBody(t *testing.T) {
	var capturedBody map[string]interface{}

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id":"page1","name":"Intro","doc_id":"doc1","workspace_id":0,"authors":[],"date_created":0,"creator_id":0,"content":""}`))
	})

	name := "Intro"
	content := "Hello world"
	cf := "text/md"
	req := &clickupv3.PublicDocsPublicCreatePageOptionsDto{
		Name:          &name,
		Content:       &content,
		ContentFormat: &cf,
	}
	page, err := apiv3.CreatePagePublic(context.Background(), client, "ws1", "doc1", req)
	require.NoError(t, err)
	assert.Equal(t, "page1", page.ID)
	assert.Equal(t, "Intro", capturedBody["name"])
	assert.Equal(t, "Hello world", capturedBody["content"])
	assert.Equal(t, "text/md", capturedBody["content_format"])
}

func TestEditPagePublic_SendsBody(t *testing.T) {
	var capturedBody map[string]interface{}

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v3/workspaces/ws1/docs/doc1/pages/page1")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(http.StatusOK)
		// API returns no body on PUT.
	})

	content := "New content"
	mode := "append"
	req := &clickupv3.PublicDocsPublicEditPageOptionsDto{
		Content:         &content,
		ContentEditMode: &mode,
	}
	err := apiv3.EditPagePublic(context.Background(), client, "ws1", "doc1", "page1", req)
	require.NoError(t, err)
	assert.Equal(t, "New content", capturedBody["content"])
	assert.Equal(t, "append", capturedBody["content_edit_mode"])
}

func TestGetPagePublic_PassesContentFormat(t *testing.T) {
	var capturedQuery string

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id":"page1","name":"Intro","doc_id":"doc1","workspace_id":0,"authors":[],"date_created":0,"creator_id":0,"content":""}`))
	})

	_, err := apiv3.GetPagePublic(context.Background(), client, "ws1", "doc1", "page1",
		apiv3.GetPagePublicParams{ContentFormat: "text/md"},
	)
	require.NoError(t, err)
	assert.Contains(t, capturedQuery, "content_format=text%2Fmd")
}

func TestGetDocPublic_APIError(t *testing.T) {
	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"err":"Doc not found","ECODE":"DOC_000"}`))
	})

	_, err := apiv3.GetDocPublic(context.Background(), client, "ws1", "notexist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestGetDocPublic_VariadicParams_NoArgs(t *testing.T) {
	var capturedURL string

	_, client := testV3Server(t, func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"docs":[],"next_cursor":null}`))
	})

	// Calling without params — should not append a query string.
	apiv3.SearchDocsPublic(context.Background(), client, "ws1")
	assert.NotContains(t, capturedURL, "?")
}
