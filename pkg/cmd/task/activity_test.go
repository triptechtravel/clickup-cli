package task

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdActivity_Usage(t *testing.T) {
	cmd := NewCmdActivity(nil)
	assert.Equal(t, "activity [<task-id>]", cmd.Use)
}

func TestParseUnixMillis(t *testing.T) {
	ts, err := parseUnixMillis("1706918400000")
	assert.NoError(t, err)
	assert.Equal(t, 2024, ts.Year())

	_, err = parseUnixMillis("invalid")
	assert.Error(t, err)
}

func TestComment_UnmarshalWithReplyCount(t *testing.T) {
	raw := `{
		"id": "123",
		"comment_text": "Parent comment",
		"user": {"username": "Alice"},
		"date": "1706918400000",
		"reply_count": 2
	}`

	var c comment
	err := json.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "123", c.ID)
	assert.Equal(t, 2, c.ReplyCount)
	assert.Nil(t, c.Replies)
}

func TestComment_MarshalIncludesReplies(t *testing.T) {
	c := comment{
		ID:          "123",
		CommentText: "Parent",
		ReplyCount:  1,
		Replies: []comment{
			{
				ID:          "456",
				CommentText: "Child reply",
				User:        commentUser{Username: "Bob"},
				Date:        "1706918500000",
			},
		},
	}

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	replies, ok := parsed["replies"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, replies, 1)
}

func TestComment_MarshalOmitsEmptyReplies(t *testing.T) {
	c := comment{
		ID:          "123",
		CommentText: "No replies",
	}

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	_, ok := parsed["replies"]
	assert.False(t, ok, "replies should be omitted when empty")
}

func TestRepliesResponse_PrefersRepliesKey(t *testing.T) {
	raw := `{
		"comments": [{"id": "c1", "comment_text": "from comments"}],
		"replies": [{"id": "r1", "comment_text": "from replies"}]
	}`

	var resp repliesResponse
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Replies, 1)
	assert.Len(t, resp.Comments, 1)
	// The fetchReplies function should prefer resp.Replies when non-empty
}

func TestRepliesResponse_FallsBackToComments(t *testing.T) {
	raw := `{
		"comments": [{"id": "c1", "comment_text": "fallback"}]
	}`

	var resp repliesResponse
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Comments, 1)
	assert.Empty(t, resp.Replies)
}
