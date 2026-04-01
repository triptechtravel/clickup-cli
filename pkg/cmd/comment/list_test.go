package comment

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdList_Usage(t *testing.T) {
	cmd := NewCmdList(nil)
	assert.Equal(t, "list [TASK]", cmd.Use)
}

func TestCommentData_UnmarshalWithReplyCount(t *testing.T) {
	raw := `{
		"id": "123",
		"comment_text": "Hello world",
		"user": {"username": "Alice"},
		"date": "1706918400000",
		"reply_count": 3
	}`

	var c commentData
	err := json.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "123", c.ID)
	assert.Equal(t, "Hello world", c.CommentText)
	assert.Equal(t, "Alice", c.User.Username)
	assert.Equal(t, 3, c.ReplyCount)
	assert.Nil(t, c.Replies)
}

func TestCommentData_UnmarshalWithoutReplyCount(t *testing.T) {
	raw := `{
		"id": "456",
		"comment_text": "No replies",
		"user": {"username": "Bob"},
		"date": "1706918400000"
	}`

	var c commentData
	err := json.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, 0, c.ReplyCount)
}

func TestCommentData_MarshalWithReplies(t *testing.T) {
	c := commentData{
		ID:          "123",
		CommentText: "Parent",
		ReplyCount:  1,
		Replies: []commentData{
			{
				ID:          "456",
				CommentText: "Reply",
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

func TestCommentData_MarshalWithoutReplies(t *testing.T) {
	c := commentData{
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

func TestCommentResponse_UnmarshalFull(t *testing.T) {
	raw := `{
		"comments": [
			{
				"id": "1",
				"comment_text": "First comment",
				"user": {"username": "Alice"},
				"date": "1706918400000",
				"reply_count": 2
			},
			{
				"id": "2",
				"comment_text": "Second comment",
				"user": {"username": "Bob"},
				"date": "1706918500000",
				"reply_count": 0
			}
		]
	}`

	var resp commentResponse
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Comments, 2)
	assert.Equal(t, 2, resp.Comments[0].ReplyCount)
	assert.Equal(t, 0, resp.Comments[1].ReplyCount)
}

func TestCommentRepliesResponse_UnmarshalRepliesKey(t *testing.T) {
	raw := `{
		"replies": [
			{
				"id": "r1",
				"comment_text": "Reply 1",
				"user": {"username": "Charlie"},
				"date": "1706918600000"
			}
		]
	}`

	var resp commentRepliesResponse
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Replies, 1)
	assert.Equal(t, "Reply 1", resp.Replies[0].CommentText)
}

func TestCommentRepliesResponse_UnmarshalCommentsKey(t *testing.T) {
	raw := `{
		"comments": [
			{
				"id": "r1",
				"comment_text": "Reply via comments key",
				"user": {"username": "Dave"},
				"date": "1706918700000"
			}
		]
	}`

	var resp commentRepliesResponse
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Comments, 1)
	assert.Empty(t, resp.Replies)
}

func TestFormatCommentDate_Valid(t *testing.T) {
	result := formatCommentDate("1706918400000")
	assert.NotEmpty(t, result)
	assert.NotEqual(t, "1706918400000", result) // should be relative time, not raw
}

func TestFormatCommentDate_Invalid(t *testing.T) {
	result := formatCommentDate("not-a-number")
	assert.Equal(t, "not-a-number", result) // falls back to raw string
}
