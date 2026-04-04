package doc

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/internal/iostreams"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

// docListResponse matches the response format from SearchDocs.
const docListJSON = `{
	"docs": [
		{"id":"doc1","name":"Project Runbook","visibility":"PUBLIC","deleted":false,"archived":false,"date_updated":"1714000000000"},
		{"id":"doc2","name":"Team Wiki","visibility":"PRIVATE","deleted":false,"archived":false,"date_updated":"1714000000000"}
	],
	"next_cursor": ""
}`

const docSingleJSON = `{
	"id": "doc1",
	"name": "Project Runbook",
	"visibility": "PUBLIC",
	"deleted": false,
	"archived": false,
	"date_created": "1714000000000",
	"date_updated": "1714100000000",
	"creator": {"id": 42, "username": "alice", "email": "alice@example.com"},
	"parent": {"id": "", "type": 0},
	"workspace": {"id": "12345"}
}`

const pageListJSON = `{
	"pages": [
		{"id":"page1","name":"Introduction","doc_id":"doc1","pages":[]},
		{"id":"page2","name":"Setup","doc_id":"doc1","pages":[
			{"id":"page3","name":"Advanced","doc_id":"doc1","pages":[]}
		]}
	]
}`

const pageSingleJSON = `{
	"id": "page1",
	"doc_id": "doc1",
	"name": "Introduction",
	"sub_title": "Getting started",
	"content": "# Hello\n\nWelcome.",
	"content_format": "text/md",
	"order_index": 0,
	"date_created": "1714000000000",
	"date_updated": "1714100000000",
	"creator": {"id": 42, "username": "alice"},
	"pages": []
}`

// v3Path returns the path under /api/v3/ for a given relative path.
func v3Path(path string) string {
	return "/api/v3/" + strings.TrimLeft(path, "/")
}

// handleV3 registers a simple JSON response handler for a v3 API path.
func handleV3(tf *testutil.TestFactory, method, path string, status int, body string) {
	fullPath := v3Path(path)
	tf.Mux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
}

// TestRunList_OutputsTable verifies that doc list renders a table with doc names.
func TestRunList_OutputsTable(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	handleV3(tf, http.MethodGet, "workspaces/12345/docs", http.StatusOK, docListJSON)

	cmd := NewCmdList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Project Runbook")
	assert.Contains(t, out, "Team Wiki")
	assert.Contains(t, out, "doc1")
	assert.Contains(t, out, "doc2")
}

// TestRunList_ParentTypeWithoutParentIDErrors validates the new error check.
func TestRunList_ParentTypeWithoutParentIDErrors(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	// Don't register any handler — validation should fail before any HTTP call.

	cmd := NewCmdList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--parent-type", "SPACE")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--parent-type requires --parent-id")
}

// TestRunList_PassesQueryParams verifies query parameters are URL-encoded correctly.
func TestRunList_PassesQueryParams(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedQuery string
	tf.Mux.HandleFunc(v3Path("workspaces/12345/docs"), func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"docs":[],"next_cursor":""}`))
	})

	cmd := NewCmdList(tf.Factory)
	err := testutil.RunCommand(t, cmd,
		"--parent-id", "sp ace+id", // intentionally has special chars
		"--parent-type", "SPACE",
		"--limit", "5",
	)
	require.NoError(t, err)
	// Query params should be properly encoded.
	assert.Contains(t, capturedQuery, "parent_id=sp+ace%2Bid")
	assert.Contains(t, capturedQuery, "parent_type=4")
	assert.Contains(t, capturedQuery, "limit=5")
}

// TestRunList_JSONOutput verifies --json outputs raw JSON.
func TestRunList_JSONOutput(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	handleV3(tf, http.MethodGet, "workspaces/12345/docs", http.StatusOK, docListJSON)

	cmd := NewCmdList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--json")
	require.NoError(t, err)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(tf.OutBuf.Bytes(), &got))
	docs := got["docs"].([]interface{})
	assert.Equal(t, 2, len(docs))
}

// TestRunView_OutputsDetails verifies doc view renders key metadata.
func TestRunView_OutputsDetails(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	handleV3(tf, http.MethodGet, "workspaces/12345/docs/doc1", http.StatusOK, docSingleJSON)

	cmd := NewCmdView(tf.Factory)
	err := testutil.RunCommand(t, cmd, "doc1")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Project Runbook")
	assert.Contains(t, out, "doc1")
	assert.Contains(t, out, "public") // visibility lowercased
	assert.Contains(t, out, "alice")  // creator username
}

// TestRunCreate_SendsCorrectBody verifies the POST body for doc create.
func TestRunCreate_SendsCorrectBody(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}
	tf.Mux.HandleFunc(v3Path("workspaces/12345/docs"), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"newdoc","name":"My Doc","visibility":"PUBLIC"}`))
	})

	cmd := NewCmdCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "My Doc", "--visibility", "PUBLIC")
	require.NoError(t, err)

	assert.Equal(t, "My Doc", capturedBody["name"])
	assert.Equal(t, "PUBLIC", capturedBody["visibility"])
	assert.Equal(t, true, capturedBody["create_page"])
}

// TestRunCreate_ParentTypeWithoutParentIDErrors validates the new error check.
func TestRunCreate_ParentTypeWithoutParentIDErrors(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	cmd := NewCmdCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "Doc", "--parent-type", "SPACE")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--parent-type requires --parent-id")
}

// TestRunPageList_PrintsTree verifies page list renders a tree.
func TestRunPageList_PrintsTree(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	handleV3(tf, http.MethodGet, "workspaces/12345/docs/doc1/pages", http.StatusOK, pageListJSON)

	cmd := NewCmdPageList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "doc1")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Introduction")
	assert.Contains(t, out, "Setup")
	assert.Contains(t, out, "Advanced")
	// Sub-page should be indented.
	assert.Contains(t, out, "  Advanced")
}

// TestPrintPageTree_Pure tests the pure printPageTree function directly.
func TestPrintPageTree_Pure(t *testing.T) {
	var buf bytes.Buffer
	pages := buildTestPageTree()

	// Use a no-color scheme so output is predictable in tests.
	cs := iostreams.NewColorScheme(false)
	printPageTree(&buf, pages, 0, cs)

	out := buf.String()
	assert.Contains(t, out, "Root Page")
	assert.Contains(t, out, "Child Page")
	// Child should be indented.
	assert.Contains(t, out, "  Child Page")
}

// TestRunPageCreate_SendsBody verifies page create POST body.
func TestRunPageCreate_SendsBody(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}
	tf.Mux.HandleFunc(v3Path("workspaces/12345/docs/doc1/pages"), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"page1","name":"Intro","doc_id":"doc1"}`))
	})

	cmd := NewCmdPageCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "doc1", "--name", "Intro", "--content", "Hello", "--content-format", "text/md")
	require.NoError(t, err)

	assert.Equal(t, "Intro", capturedBody["name"])
	assert.Equal(t, "Hello", capturedBody["content"])
	assert.Equal(t, "text/md", capturedBody["content_format"])
}

// TestRunPageEdit_SendsBody verifies page edit PUT body.
func TestRunPageEdit_SendsBody(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}
	tf.Mux.HandleFunc(v3Path("workspaces/12345/docs/doc1/pages/page1"), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"page1","name":"Updated","doc_id":"doc1"}`))
	})

	cmd := NewCmdPageEdit(tf.Factory)
	err := testutil.RunCommand(t, cmd, "doc1", "page1",
		"--content", "New content",
		"--content-edit-mode", "append",
	)
	require.NoError(t, err)

	assert.Equal(t, "New content", capturedBody["content"])
	assert.Equal(t, "append", capturedBody["content_edit_mode"])
}

// TestRunPageView_OutputsContent verifies page view renders content.
func TestRunPageView_OutputsContent(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	handleV3(tf, http.MethodGet, "workspaces/12345/docs/doc1/pages/page1", http.StatusOK, pageSingleJSON)

	cmd := NewCmdPageView(tf.Factory)
	err := testutil.RunCommand(t, cmd, "doc1", "page1")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Introduction")
	assert.Contains(t, out, "Getting started")
	assert.Contains(t, out, "# Hello")
	assert.Contains(t, out, "alice")
}

// TestRunPageView_ContentFormatQueryParam verifies content_format is passed.
func TestRunPageView_ContentFormatQueryParam(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedQuery string
	tf.Mux.HandleFunc(v3Path("workspaces/12345/docs/doc1/pages/page1"), func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(pageSingleJSON))
	})

	cmd := NewCmdPageView(tf.Factory)
	err := testutil.RunCommand(t, cmd, "doc1", "page1", "--content-format", "text/md")
	require.NoError(t, err)

	assert.Contains(t, capturedQuery, "content_format=text%2Fmd")
}

// buildTestPageTree returns a minimal page tree for pure function tests.
func buildTestPageTree() []apiv3.PageRef {
	return []apiv3.PageRef{
		{
			ID:   "root1",
			Name: "Root Page",
			Pages: []apiv3.PageRef{
				{ID: "child1", Name: "Child Page"},
			},
		},
	}
}
