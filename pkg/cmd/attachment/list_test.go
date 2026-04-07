package attachment

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

var sampleAttachmentsJSON = `{
	"data": [
		{
			"id": "att_001",
			"title": "screenshot.png",
			"extension": "png",
			"mime_type": "image/png",
			"size": 204800,
			"url": "https://attachments.clickup.com/screenshot.png",
			"parent_entity_type": "tasks",
			"parent_id": "abc123",
			"date_created": 1700000000000,
			"date_updated": 1700100000000,
			"user_id": 1,
			"signed": false,
			"thumbnail_small": "",
			"thumbnail_medium": "",
			"thumbnail_large": ""
		},
		{
			"id": "att_002",
			"title": "report.pdf",
			"extension": "pdf",
			"mime_type": "application/pdf",
			"size": 1048576,
			"url": "https://attachments.clickup.com/report.pdf",
			"parent_entity_type": "tasks",
			"parent_id": "abc123",
			"date_created": 1700000000000,
			"date_updated": 1700100000000,
			"user_id": 1,
			"signed": false,
			"thumbnail_small": "",
			"thumbnail_medium": "",
			"thumbnail_large": ""
		}
	],
	"cursor": ""
}`

func TestListCommand_Table(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleV3("GET", "workspaces/12345/attachments/abc123/attachments", 200, sampleAttachmentsJSON)

	cmd := NewCmdList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "screenshot.png")
	assert.Contains(t, out, "report.pdf")
	assert.Contains(t, out, "png")
	assert.Contains(t, out, "pdf")
}

func TestListCommand_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleV3("GET", "workspaces/12345/attachments/abc123/attachments", 200, sampleAttachmentsJSON)

	cmd := NewCmdList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed), "output should be valid JSON")
	assert.Len(t, parsed, 2)
	assert.Equal(t, "att_001", parsed[0]["id"])
	assert.Equal(t, "screenshot.png", parsed[0]["title"])
}

func TestListCommand_Empty(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	tf.HandleV3("GET", "workspaces/12345/attachments/abc123/attachments", 200, `{"data": [], "cursor": ""}`)

	cmd := NewCmdList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123")
	require.NoError(t, err)

	assert.Contains(t, tf.ErrBuf.String(), "No attachments found")
}
