package task

import (
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

func TestChecklistItemEdit_AssigneeDoesNotWipeName(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// Track what the PUT request body contains — this is the critical assertion.
	var capturedBody map[string]interface{}
	var putCalled atomic.Int32

	tf.HandleFunc("checklist/cl-1/checklist_item/item-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")

		if r.Method == "PUT" {
			putCalled.Add(1)
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.Write([]byte(`{"checklist": {}}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	cmd := newCmdChecklistItemEdit(tf.Factory)
	err := testutil.RunCommand(t, cmd, "cl-1", "item-1", "--assignee", "12345")
	require.NoError(t, err)

	assert.Equal(t, int32(1), putCalled.Load())

	// The request body must NOT contain a "name" key at all.
	_, hasName := capturedBody["name"]
	assert.False(t, hasName, "setting --assignee should not send a name field (would wipe it)")

	// The assignee should be present.
	assert.Contains(t, capturedBody, "assignee")
}

func TestChecklistItemEdit_NameDoesNotWipeAssignee(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}

	tf.HandleFunc("checklist/cl-1/checklist_item/item-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")

		if r.Method == "PUT" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.Write([]byte(`{"checklist": {}}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	cmd := newCmdChecklistItemEdit(tf.Factory)
	err := testutil.RunCommand(t, cmd, "cl-1", "item-1", "--name", "New name")
	require.NoError(t, err)

	// Should have name but NOT assignee.
	assert.Equal(t, "New name", capturedBody["name"])
	_, hasAssignee := capturedBody["assignee"]
	assert.False(t, hasAssignee, "setting --name should not send an assignee field")
}

func TestChecklistItemEdit_BothNameAndAssignee(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}

	tf.HandleFunc("checklist/cl-1/checklist_item/item-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")

		if r.Method == "PUT" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.Write([]byte(`{"checklist": {}}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	cmd := newCmdChecklistItemEdit(tf.Factory)
	err := testutil.RunCommand(t, cmd, "cl-1", "item-1", "--name", "Updated", "--assignee", "99")
	require.NoError(t, err)

	assert.Equal(t, "Updated", capturedBody["name"])
	assert.Contains(t, capturedBody, "assignee")
}

func TestChecklistItemEdit_BulkAssignee(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var putCount atomic.Int32
	var bodies []map[string]interface{}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")

		if r.Method == "PUT" {
			putCount.Add(1)
			body, _ := io.ReadAll(r.Body)
			var b map[string]interface{}
			json.Unmarshal(body, &b)
			bodies = append(bodies, b)
			w.Write([]byte(`{"checklist": {}}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	tf.HandleFunc("checklist/cl-1/checklist_item/item-1", handler)
	tf.HandleFunc("checklist/cl-1/checklist_item/item-2", handler)
	tf.HandleFunc("checklist/cl-1/checklist_item/item-3", handler)

	cmd := newCmdChecklistItemEdit(tf.Factory)
	err := testutil.RunCommand(t, cmd, "cl-1", "item-1", "item-2", "item-3", "--assignee", "555")
	require.NoError(t, err)

	assert.Equal(t, int32(3), putCount.Load())

	// None of the requests should contain a "name" field.
	for i, b := range bodies {
		_, hasName := b["name"]
		assert.False(t, hasName, "request %d should not have name field", i)
	}

	out := tf.OutBuf.String()
	assert.Contains(t, out, "3/3")
}

func TestChecklistItemEdit_RequiresFlag(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	cmd := newCmdChecklistItemEdit(tf.Factory)
	err := testutil.RunCommand(t, cmd, "cl-1", "item-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one of --assignee or --name")
}

func TestChecklistItemEdit_BulkNameRejected(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	cmd := newCmdChecklistItemEdit(tf.Factory)
	err := testutil.RunCommand(t, cmd, "cl-1", "item-1", "item-2", "--name", "foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--name can only be used with a single item")
}

func TestChecklistItemResolve_DoesNotWipeName(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}

	tf.HandleFunc("checklist/cl-1/checklist_item/item-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")

		if r.Method == "PUT" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.Write([]byte(`{"checklist": {}}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	cmd := newCmdChecklistItemResolve(tf.Factory)
	err := testutil.RunCommand(t, cmd, "cl-1", "item-1")
	require.NoError(t, err)

	// Must have resolved=true but NOT name.
	assert.Equal(t, true, capturedBody["resolved"])
	_, hasName := capturedBody["name"]
	assert.False(t, hasName, "resolve should not send a name field (would wipe it)")
}

// Regression: ClickUp returns the checklist item's `assignee` as a full user
// object (not the string the raw spec declares). The response must decode
// without error. See fix_checklist_item_assignee in patch-v2-spec.jq.
func TestChecklistItemResolve_DecodesObjectAssignee(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleFunc("checklist/cl-1/checklist_item/item-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")

		if r.Method == "PUT" {
			// assignee is an object here — the shape the live API returns for
			// an assigned item. Before the spec patch this failed to decode.
			w.Write([]byte(`{"checklist": {"items": [{"id": "item-1", "name": "Do the thing", "orderindex": 0, "assignee": {"id": 123, "username": "Isaac", "email": "isaac@triptech.com"}, "resolved": true, "parent": null, "date_created": "1700000000000", "children": []}]}}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	cmd := newCmdChecklistItemResolve(tf.Factory)
	err := testutil.RunCommand(t, cmd, "cl-1", "item-1")
	require.NoError(t, err, "object-shaped assignee in the response must decode")
}

func TestChecklistItemAdd_WithAssignee(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}

	tf.HandleFunc("checklist/cl-1/checklist_item", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")

		if r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.Write([]byte(`{"checklist": {}}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	cmd := newCmdChecklistItemAdd(tf.Factory)
	err := testutil.RunCommand(t, cmd, "cl-1", "My item", "--assignee", "42")
	require.NoError(t, err)

	assert.Equal(t, "My item", capturedBody["name"])
	assert.Equal(t, float64(42), capturedBody["assignee"])
}

func TestChecklistItemAdd_WithoutAssignee(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}

	tf.HandleFunc("checklist/cl-1/checklist_item", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")

		if r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.Write([]byte(`{"checklist": {}}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	cmd := newCmdChecklistItemAdd(tf.Factory)
	err := testutil.RunCommand(t, cmd, "cl-1", "My item")
	require.NoError(t, err)

	assert.Equal(t, "My item", capturedBody["name"])
	_, hasAssignee := capturedBody["assignee"]
	assert.False(t, hasAssignee, "no --assignee flag should omit the field")
}
