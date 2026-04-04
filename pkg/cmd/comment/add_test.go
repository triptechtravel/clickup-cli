package comment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeMembers(entries ...struct{ username string; id int }) map[string]workspaceMember {
	m := make(map[string]workspaceMember)
	for _, e := range entries {
		m[e.username] = workspaceMember{Username: e.username, ID: e.id}
	}
	return m
}

// helper to create members map more conveniently
func membersMap(pairs ...interface{}) map[string]workspaceMember {
	m := make(map[string]workspaceMember)
	for i := 0; i < len(pairs); i += 2 {
		username := pairs[i].(string)
		id := pairs[i+1].(int)
		m[username] = workspaceMember{Username: username, ID: id}
	}
	return m
}

// ---------------------------------------------------------------------------
// buildCommentBlocks
// ---------------------------------------------------------------------------

func TestBuildCommentBlocks(t *testing.T) {
	members := membersMap(
		"isaac", 100,
		"alice", 200,
		"bob", 300,
		"isaacrowntree", 400,
	)

	tests := []struct {
		name             string
		body             string
		members          map[string]workspaceMember
		wantResolved     []string
		wantBlockCount   int
		wantNilBlocks    bool
	}{
		{
			name:           "single mention",
			body:           "Hey @alice check this",
			members:        members,
			wantResolved:   []string{"alice"},
			wantBlockCount: 3, // "Hey " + tag + " check this"
		},
		{
			name:           "multiple mentions",
			body:           "@alice and @bob please review",
			members:        members,
			wantResolved:   []string{"alice", "bob"},
			wantBlockCount: 4, // tag + " and " + tag + " please review"
		},
		{
			name:           "adjacent mentions",
			body:           "@alice@bob done",
			members:        members,
			wantResolved:   []string{"alice", "bob"},
			wantBlockCount: 3, // tag + tag + " done" (@ is not a word char so both resolve)
		},
		{
			name:           "mention at start",
			body:           "@alice hi",
			members:        members,
			wantResolved:   []string{"alice"},
			wantBlockCount: 2, // tag + " hi"
		},
		{
			name:           "mention at end",
			body:           "Thanks @bob",
			members:        members,
			wantResolved:   []string{"bob"},
			wantBlockCount: 2, // "Thanks " + tag
		},
		{
			name:          "no match",
			body:          "Hello @nobody here",
			members:       members,
			wantNilBlocks: true,
		},
		{
			name:           "word boundary check - @isaac should not match @isaacrowntree",
			body:           "Hey @isaacrowntree look",
			members:        members,
			wantResolved:   []string{"isaacrowntree"},
			wantBlockCount: 3,
		},
		{
			name:           "overlapping usernames longer wins",
			body:           "Hey @isaacrowntree and @isaac check",
			members:        members,
			wantResolved:   []string{"isaacrowntree", "isaac"},
			wantBlockCount: 5, // "Hey " + tag + " and " + tag + " check"
		},
		{
			name:           "case-insensitive",
			body:           "Hey @Alice check",
			members:        members,
			wantResolved:   []string{"alice"},
			wantBlockCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, resolved := buildCommentBlocks(tt.body, tt.members)
			if tt.wantNilBlocks {
				assert.Nil(t, blocks)
				assert.Nil(t, resolved)
				return
			}
			require.NotNil(t, blocks)
			assert.Equal(t, tt.wantResolved, resolved)
			assert.Len(t, blocks, tt.wantBlockCount)
		})
	}
}

// ---------------------------------------------------------------------------
// isWordChar
// ---------------------------------------------------------------------------

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		name string
		char byte
		want bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'Z', true},
		{"digit", '5', true},
		{"underscore", '_', true},
		{"space", ' ', false},
		{"at sign", '@', false},
		{"hyphen", '-', false},
		{"period", '.', false},
		{"exclamation", '!', false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isWordChar(tt.char))
		})
	}
}
