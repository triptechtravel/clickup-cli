package chat

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

func TestNewCmdChat_HasSendSubcommand(t *testing.T) {
	cmd := NewCmdChat(nil)
	assert.Equal(t, "chat <command>", cmd.Use)

	sub, _, err := cmd.Find([]string{"send"})
	require.NoError(t, err)
	assert.Equal(t, "send", sub.Name())
}

func TestChatSend_SendsCorrectRequest(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}
	var capturedMethod string
	var capturedPath string

	tf.HandleFuncV3("workspaces/12345/chat/channels/chan-abc/messages", func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"content": "Hello team!",
			"type": "message",
			"date": 1700000000000,
			"user_id": "user1",
			"resolved": false,
			"links": {"reactions": "", "replies": "", "tagged_users": ""},
			"replies_count": 0
		}`))
	})

	cmd := NewCmdSend(tf.Factory)
	err := testutil.RunCommand(t, cmd, "chan-abc", "Hello team!")
	require.NoError(t, err)

	assert.Equal(t, "POST", capturedMethod)
	assert.Contains(t, capturedPath, "/workspaces/12345/chat/channels/chan-abc/messages")
	assert.Equal(t, "message", capturedBody["type"])
	assert.Equal(t, "Hello team!", capturedBody["content"])
	assert.Equal(t, "text/md", capturedBody["content_format"])

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Message sent")
	assert.Contains(t, out, "chan-abc")
}

func TestChatSend_JSONOutput(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleFuncV3("workspaces/12345/chat/channels/chan-abc/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"content": "Hi",
			"type": "message",
			"date": 1700000000000,
			"user_id": "user1",
			"resolved": false,
			"links": {"reactions": "", "replies": "", "tagged_users": ""},
			"replies_count": 0
		}`))
	})

	cmd := NewCmdSend(tf.Factory)
	err := testutil.RunCommand(t, cmd, "chan-abc", "Hi", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	// Should be valid JSON
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "Hi", parsed["content"])
}

func TestChatSend_RequiresArgs(t *testing.T) {
	cmd := NewCmdSend(nil)
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"chan-only"}))
	assert.NoError(t, cmd.Args(cmd, []string{"chan-id", "message"}))
	assert.Error(t, cmd.Args(cmd, []string{"a", "b", "c"}))
}

// ── chat list ────────────────────────────────────────────────────────

func TestNewCmdChat_HasAllSubcommands(t *testing.T) {
	cmd := NewCmdChat(nil)

	for _, name := range []string{"send", "list", "messages", "reply", "react", "delete"} {
		sub, _, err := cmd.Find([]string{name})
		require.NoError(t, err)
		assert.Equal(t, name, sub.Name())
	}
}

func TestChatList_ReturnsTable(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleV3("GET", "workspaces/12345/chat/channels", 200, `{
		"next_cursor": "",
		"data": [
			{"id":"ch1","name":"General","type":"PUBLIC","visibility":"PUBLIC","parent":{},"creator":"u1","created_at":"","updated_at":"","workspace_id":"12345","archived":false},
			{"id":"ch2","name":"Random","type":"DIRECT_MESSAGE","visibility":"PRIVATE","parent":{},"creator":"u2","created_at":"","updated_at":"","workspace_id":"12345","archived":false}
		]
	}`)

	cmd := NewCmdList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "ch1")
	assert.Contains(t, out, "General")
	assert.Contains(t, out, "ch2")
	assert.Contains(t, out, "Random")
}

func TestChatList_JSONOutput(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleV3("GET", "workspaces/12345/chat/channels", 200, `{
		"next_cursor": "",
		"data": [
			{"id":"ch1","name":"General","type":"PUBLIC","visibility":"PUBLIC","parent":{},"creator":"u1","created_at":"","updated_at":"","workspace_id":"12345","archived":false}
		]
	}`)

	cmd := NewCmdList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	data := parsed["data"].([]interface{})
	assert.Len(t, data, 1)
	assert.Equal(t, "General", data[0].(map[string]interface{})["name"])
}

func TestChatList_EmptyResult(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleV3("GET", "workspaces/12345/chat/channels", 200, `{
		"next_cursor": "",
		"data": []
	}`)

	cmd := NewCmdList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "No channels found")
}

// ── chat messages ────────────────────────────────────────────────────

func TestChatMessages_ReturnsTable(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleV3("GET", "workspaces/12345/chat/channels/chan-abc/messages", 200, `{
		"next_cursor": "",
		"data": [
			{
				"id":"msg1",
				"user_id":"user1",
				"content":"Hello world",
				"date":1700000000000,
				"type":"message",
				"parent_channel":"chan-abc",
				"resolved":false,
				"links":{"reactions":"","replies":"","tagged_users":""},
				"replies_count":0
			},
			{
				"id":"msg2",
				"user_id":"user2",
				"content":"Goodbye world",
				"date":1700000001000,
				"type":"message",
				"parent_channel":"chan-abc",
				"resolved":false,
				"links":{"reactions":"","replies":"","tagged_users":""},
				"replies_count":0
			}
		]
	}`)

	cmd := NewCmdMessages(tf.Factory)
	err := testutil.RunCommand(t, cmd, "chan-abc")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "msg1")
	assert.Contains(t, out, "Hello world")
	assert.Contains(t, out, "msg2")
	assert.Contains(t, out, "Goodbye world")
}

func TestChatMessages_JSONOutput(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleV3("GET", "workspaces/12345/chat/channels/chan-abc/messages", 200, `{
		"next_cursor": "",
		"data": [
			{
				"id":"msg1",
				"user_id":"user1",
				"content":"Hello",
				"date":1700000000000,
				"type":"message",
				"parent_channel":"chan-abc",
				"resolved":false,
				"links":{"reactions":"","replies":"","tagged_users":""},
				"replies_count":0
			}
		]
	}`)

	cmd := NewCmdMessages(tf.Factory)
	err := testutil.RunCommand(t, cmd, "chan-abc", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	data := parsed["data"].([]interface{})
	assert.Len(t, data, 1)
	assert.Equal(t, "Hello", data[0].(map[string]interface{})["content"])
}

func TestChatMessages_RequiresChannelArg(t *testing.T) {
	cmd := NewCmdMessages(nil)
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.NoError(t, cmd.Args(cmd, []string{"chan-id"}))
}

// ── chat reply ──────────────────────────────────────────────────────

func TestChatReply(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}
	var capturedMethod string
	var capturedPath string

	tf.HandleFuncV3("workspaces/12345/chat/messages/msg-abc/replies", func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"content": "Got it!",
			"type": "message",
			"date": 1700000000000,
			"user_id": "user1",
			"resolved": false,
			"links": {"reactions": "", "replies": "", "tagged_users": ""},
			"replies_count": 0
		}`))
	})

	cmd := NewCmdReply(tf.Factory)
	err := testutil.RunCommand(t, cmd, "msg-abc", "Got it!")
	require.NoError(t, err)

	assert.Equal(t, "POST", capturedMethod)
	assert.Contains(t, capturedPath, "/workspaces/12345/chat/messages/msg-abc/replies")
	assert.Equal(t, "message", capturedBody["type"])
	assert.Equal(t, "Got it!", capturedBody["content"])

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Reply sent")
	assert.Contains(t, out, "msg-abc")
}

// ── chat react ──────────────────────────────────────────────────────

func TestChatReact(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedBody map[string]interface{}
	var capturedMethod string
	var capturedPath string

	tf.HandleFuncV3("workspaces/12345/chat/messages/msg-abc/reactions", func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	})

	cmd := NewCmdReact(tf.Factory)
	err := testutil.RunCommand(t, cmd, "msg-abc", "rocket")
	require.NoError(t, err)

	assert.Equal(t, "POST", capturedMethod)
	assert.Contains(t, capturedPath, "/workspaces/12345/chat/messages/msg-abc/reactions")
	assert.Equal(t, "rocket", capturedBody["reaction"])

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Reaction")
	assert.Contains(t, out, "rocket")
	assert.Contains(t, out, "msg-abc")
}

// ── chat delete ─────────────────────────────────────────────────────

func TestChatDelete(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	var capturedMethod string
	var capturedPath string

	tf.HandleFuncV3("workspaces/12345/chat/messages/msg-del", func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	})

	cmd := NewCmdDelete(tf.Factory)
	err := testutil.RunCommand(t, cmd, "msg-del", "--yes")
	require.NoError(t, err)

	assert.Equal(t, "DELETE", capturedMethod)
	assert.Contains(t, capturedPath, "/workspaces/12345/chat/messages/msg-del")

	out := tf.OutBuf.String()
	assert.Contains(t, out, "deleted")
	assert.Contains(t, out, "msg-del")
}
