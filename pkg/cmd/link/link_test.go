package link

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdLinkPR_Flags(t *testing.T) {
	cmd := NewCmdLinkPR(nil)

	assert.NotNil(t, cmd.Flags().Lookup("task"))
	assert.NotNil(t, cmd.Flags().Lookup("repo"))
	assert.Equal(t, "pr [NUMBER]", cmd.Use)
}

func TestNewCmdLinkBranch_Flags(t *testing.T) {
	cmd := NewCmdLinkBranch(nil)

	assert.NotNil(t, cmd.Flags().Lookup("task"))
	assert.Equal(t, "branch", cmd.Use)
}

func TestNewCmdLinkSync_Flags(t *testing.T) {
	cmd := NewCmdLinkSync(nil)

	assert.NotNil(t, cmd.Flags().Lookup("task"))
	assert.NotNil(t, cmd.Flags().Lookup("repo"))
	assert.Equal(t, "sync [PR-NUMBER]", cmd.Use)
}

func TestInferRepoFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{
			url:  "https://github.com/triptechtravel/campermate.com/pull/1109",
			want: "triptechtravel/campermate.com",
		},
		{
			url:  "https://github.com/owner/repo/pull/42",
			want: "owner/repo",
		},
		{
			url:  "",
			want: "",
		},
		{
			url:  "not-a-url",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := inferRepoFromURL(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpsertClickUpBlock(t *testing.T) {
	block := "<!-- clickup-cli:start -->\ntest\n<!-- clickup-cli:end -->"

	t.Run("empty body", func(t *testing.T) {
		result := upsertClickUpBlock("", block)
		assert.Equal(t, block, result)
	})

	t.Run("prepends to existing body", func(t *testing.T) {
		result := upsertClickUpBlock("existing content", block)
		assert.Contains(t, result, block)
		assert.Contains(t, result, "existing content")
	})

	t.Run("replaces existing block", func(t *testing.T) {
		oldBody := "<!-- clickup-cli:start -->\nold\n<!-- clickup-cli:end -->\n\nother content"
		result := upsertClickUpBlock(oldBody, block)
		assert.Contains(t, result, "test")
		assert.NotContains(t, result, "old")
		assert.Contains(t, result, "other content")
	})
}

func TestBuildClickUpBlock(t *testing.T) {
	block := buildClickUpBlock(
		"https://app.clickup.com/t/abc123",
		"Test Task",
		"in progress",
		"high",
		[]string{"Isaac", "Bob"},
	)

	assert.Contains(t, block, clickupBlockStart)
	assert.Contains(t, block, clickupBlockEnd)
	assert.Contains(t, block, "Test Task")
	assert.Contains(t, block, "in progress")
	assert.Contains(t, block, "high")
	assert.Contains(t, block, "Isaac, Bob")
}
