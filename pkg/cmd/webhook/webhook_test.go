package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

var sampleWebhooksJSON = `{
	"webhooks": [
		{
			"id": "wh-1",
			"userid": 1,
			"team_id": 12345,
			"endpoint": "https://example.com/hook1",
			"client_id": "client1",
			"events": ["taskCreated", "taskUpdated"],
			"task_id": null,
			"list_id": null,
			"folder_id": null,
			"space_id": null,
			"health": {"status": "active", "fail_count": 0},
			"secret": "secret123"
		},
		{
			"id": "wh-2",
			"userid": 1,
			"team_id": 12345,
			"endpoint": "https://example.com/hook2",
			"client_id": "client2",
			"events": ["*"],
			"task_id": null,
			"list_id": null,
			"folder_id": null,
			"space_id": null,
			"health": null,
			"secret": "secret456"
		}
	]
}`

func webhooksHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(body))
	}
}

func TestWebhookList(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/webhook", webhooksHandler(sampleWebhooksJSON))

	cmd := NewCmdWebhookList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "wh-1")
	assert.Contains(t, out, "https://example.com/hook1")
	assert.Contains(t, out, "wh-2")
	assert.Contains(t, out, "https://example.com/hook2")
}

func TestWebhookList_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/webhook", webhooksHandler(sampleWebhooksJSON))

	cmd := NewCmdWebhookList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed), "output should be valid JSON")
	assert.Len(t, parsed, 2)
	assert.Equal(t, "wh-1", parsed[0]["id"])
}

func TestWebhookList_Empty(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/webhook", webhooksHandler(`{"webhooks": []}`))

	cmd := NewCmdWebhookList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "No webhooks found.")
}

func TestWebhookCreate(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/webhook", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "https://example.com/hook", req["endpoint"])

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"id": "wh-new", "webhook": {}}`))
	})

	cmd := NewCmdWebhookCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--endpoint", "https://example.com/hook", "--events", "taskCreated")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Webhook created")
	assert.Contains(t, out, "wh-new")
}
