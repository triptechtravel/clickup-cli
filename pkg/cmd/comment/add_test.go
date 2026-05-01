package comment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
// resolveMentionsInBlocks
// ---------------------------------------------------------------------------

func TestResolveMentionsInBlocks(t *testing.T) {
	members := membersMap(
		"alex", 100,
		"alice", 200,
		"bob", 300,
		"alexanderx", 400,
	)

	tests := []struct {
		name           string
		body           string
		wantResolved   []string
		wantBlockCount int
	}{
		{"single mention", "Hey @alice check this", []string{"alice"}, 3},
		{"multiple mentions", "@alice and @bob please review", []string{"alice", "bob"}, 4},
		{"adjacent mentions", "@alice@bob done", []string{"alice", "bob"}, 3},
		{"mention at start", "@alice hi", []string{"alice"}, 2},
		{"mention at end", "Thanks @bob", []string{"bob"}, 2},
		{"word boundary - @alex in @alexanderx", "Hey @alexanderx look", []string{"alexanderx"}, 3},
		{"longest-first wins", "Hey @alexanderx and @alex check", []string{"alexanderx", "alex"}, 5},
		{"case-insensitive", "Hey @Alice check", []string{"alice"}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, resolved := resolveMentionsInBlocks([]commentBlock{{Text: tt.body}}, members)
			assert.Equal(t, tt.wantResolved, resolved)
			assert.Len(t, blocks, tt.wantBlockCount)
		})
	}
}

func TestResolveMentionsInBlocks_NoMatch(t *testing.T) {
	members := membersMap("alice", 100)
	in := []commentBlock{{Text: "hello @nobody here"}}
	out, resolved := resolveMentionsInBlocks(in, members)
	assert.Nil(t, resolved)
	assert.Equal(t, in, out)
}

func TestResolveMentionsInBlocks_PreservesAttributes(t *testing.T) {
	members := membersMap("alice", 100)
	in := []commentBlock{{Text: "hi @alice", Attributes: map[string]interface{}{"bold": true}}}
	out, _ := resolveMentionsInBlocks(in, members)
	require.Len(t, out, 2)
	assert.Equal(t, "hi ", out[0].Text)
	assert.Equal(t, true, out[0].Attributes["bold"])
	assert.Equal(t, "tag", out[1].Type)
	assert.Nil(t, out[1].Attributes)
}

// ---------------------------------------------------------------------------
// markdownToBlocks
// ---------------------------------------------------------------------------

func TestMarkdownToBlocks_Headings(t *testing.T) {
	blocks := markdownToBlocks("## Hello\n\nWorld")
	require.GreaterOrEqual(t, len(blocks), 3)
	assert.Equal(t, "Hello", blocks[0].Text)
	assert.Equal(t, "\n", blocks[1].Text)
	assert.Equal(t, 2, blocks[1].Attributes["header"])
}

func TestMarkdownToBlocks_BoldItalicCode(t *testing.T) {
	blocks := markdownToBlocks("**bold** and *italic* with `code`")
	var got []string
	attrs := map[string]map[string]interface{}{}
	for _, b := range blocks {
		got = append(got, b.Text)
		if b.Attributes != nil {
			attrs[b.Text] = b.Attributes
		}
	}
	assert.Contains(t, got, "bold")
	assert.True(t, attrs["bold"]["bold"].(bool))
	assert.True(t, attrs["italic"]["italic"].(bool))
	assert.True(t, attrs["code"]["code"].(bool))
}

func TestMarkdownToBlocks_BulletList(t *testing.T) {
	blocks := markdownToBlocks("- one\n- two")
	bulletNewlines := 0
	for _, b := range blocks {
		if b.Text == "\n" && b.Attributes["list"] == "bullet" {
			bulletNewlines++
		}
	}
	assert.Equal(t, 2, bulletNewlines)
}

func TestMarkdownToBlocks_OrderedList(t *testing.T) {
	blocks := markdownToBlocks("1. first\n2. second")
	orderedNewlines := 0
	for _, b := range blocks {
		if b.Text == "\n" && b.Attributes["list"] == "ordered" {
			orderedNewlines++
		}
	}
	assert.Equal(t, 2, orderedNewlines)
}

func TestMarkdownToBlocks_NestedListInsideOrdered(t *testing.T) {
	src := "1. outer one\n   - inner a\n   - inner b\n2. outer two"
	blocks := markdownToBlocks(src)

	// Walk through and confirm each item's terminating \n carries the right
	// list type and indent. Order matters: outer-1, inner-a, inner-b, outer-2.
	type marker struct {
		list   string
		indent int
	}
	var markers []marker
	for _, b := range blocks {
		if b.Text != "\n" {
			continue
		}
		l, _ := b.Attributes["list"].(string)
		if l == "" {
			continue
		}
		ind, _ := b.Attributes["indent"].(int)
		markers = append(markers, marker{l, ind})
	}
	assert.Equal(t, []marker{
		{"ordered", 0},
		{"bullet", 1},
		{"bullet", 1},
		{"ordered", 0},
	}, markers)
}

func TestMarkdownToBlocks_NestedListSeparatesParentText(t *testing.T) {
	// The parent item's text must not bleed into the first nested item — it
	// must be terminated by the parent's `list: ordered` newline first.
	blocks := markdownToBlocks("1. parent text:\n   - child item")
	var seenOrderedTerminator bool
	var sawChildBeforeTerminator bool
	for _, b := range blocks {
		if !seenOrderedTerminator {
			if b.Text == "\n" && b.Attributes["list"] == "ordered" {
				seenOrderedTerminator = true
				continue
			}
			if strings.Contains(b.Text, "child item") {
				sawChildBeforeTerminator = true
			}
		}
	}
	assert.True(t, seenOrderedTerminator, "ordered newline should appear")
	assert.False(t, sawChildBeforeTerminator, "child text must come after parent's terminator")
}

func TestMarkdownToBlocks_Link(t *testing.T) {
	blocks := markdownToBlocks("see [docs](https://example.com)")
	var linkBlock *commentBlock
	for i, b := range blocks {
		if b.Attributes["link"] != nil {
			linkBlock = &blocks[i]
			break
		}
	}
	require.NotNil(t, linkBlock)
	assert.Equal(t, "docs", linkBlock.Text)
	assert.Equal(t, "https://example.com", linkBlock.Attributes["link"])
}

func TestMarkdownToBlocks_FencedCodeBlock(t *testing.T) {
	blocks := markdownToBlocks("```\nline1\nline2\n```")
	codeNewlines := 0
	for _, b := range blocks {
		if b.Attributes["code-block"] == true {
			codeNewlines++
		}
	}
	assert.GreaterOrEqual(t, codeNewlines, 2)
}

// ---------------------------------------------------------------------------
// buildBlocks
// ---------------------------------------------------------------------------

func TestBuildBlocks_PlainTextDoesNotNeedBlocks(t *testing.T) {
	_, resolved, useBlocks := buildBlocks("just plain text", nil)
	assert.Nil(t, resolved)
	assert.False(t, useBlocks)
}

func TestBuildBlocks_FormattedNeedsBlocks(t *testing.T) {
	_, resolved, useBlocks := buildBlocks("**bold** text", nil)
	assert.Nil(t, resolved)
	assert.True(t, useBlocks)
}

func TestBuildBlocks_MentionNeedsBlocks(t *testing.T) {
	members := membersMap("alice", 100)
	_, resolved, useBlocks := buildBlocks("hi @alice", members)
	assert.Equal(t, []string{"alice"}, resolved)
	assert.True(t, useBlocks)
}

func TestBuildBlocks_MentionInsideHeader(t *testing.T) {
	members := membersMap("alice", 100)
	blocks, resolved, useBlocks := buildBlocks("## @alice review", members)
	assert.Equal(t, []string{"alice"}, resolved)
	assert.True(t, useBlocks)
	var hasTag, hasHeader bool
	for _, b := range blocks {
		if b.Type == "tag" {
			hasTag = true
		}
		if b.Attributes["header"] == 1 || b.Attributes["header"] == 2 {
			hasHeader = true
		}
	}
	assert.True(t, hasTag)
	assert.True(t, hasHeader)
}

// ---------------------------------------------------------------------------
// toCreateBlocks / toUpdateBlocks / toReplyMap
// ---------------------------------------------------------------------------

func TestToCreateBlocks_RoundtripsTextTypeUserAttributes(t *testing.T) {
	in := []commentBlock{
		{Text: "hi "},
		{Type: "tag", User: &mentionUser{ID: 42}},
		{Text: "bold", Attributes: map[string]interface{}{"bold": true}},
	}
	out := toCreateBlocks(in)
	require.Len(t, out, 3)
	require.NotNil(t, out[0].Text)
	assert.Equal(t, "hi ", *out[0].Text)
	require.NotNil(t, out[1].Type)
	assert.Equal(t, "tag", *out[1].Type)
	require.NotNil(t, out[1].User)
	require.NotNil(t, out[1].User.ID)
	assert.Equal(t, 42, *out[1].User.ID)
	assert.Equal(t, true, out[2].Attributes["bold"])
}

func TestToReplyMap_OmitsZeroFields(t *testing.T) {
	in := []commentBlock{
		{Text: "hello"},
		{Type: "tag", User: &mentionUser{ID: 7}},
	}
	out := toReplyMap(in)
	require.Len(t, out, 2)
	assert.Equal(t, "hello", out[0]["text"])
	assert.NotContains(t, out[0], "type")
	assert.NotContains(t, out[0], "user")
	assert.Equal(t, "tag", out[1]["type"])
	assert.NotContains(t, out[1], "text")
}

// ---------------------------------------------------------------------------
// aliasKeys / first-name resolution
// ---------------------------------------------------------------------------

func TestAliasKeys(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		want     []string
	}{
		// Multi-word username with first-name == email local-part: dedup'd to
		// a single key so the ambiguity counter sees exactly one occurrence
		// per user.
		{"first-name matches email local", "First Last", "first@example.com", []string{"first"}},
		// Email with dotted local-part contributes both the dotted prefix and
		// the full local-part, plus first-name token if multi-word.
		{"dotted email + multi-word username", "Alpha Beta", "alpha.beta@example.com", []string{"alpha", "alpha.beta"}},
		// Single-token username without first-name token; only email aliases.
		{"single-token username", "singletoken", "x@y.com", []string{"x"}},
		// No email at all — nothing usable.
		{"no email, single-token", "NoEmail", "", nil},
		// Multi-word username, no email — first-name token only.
		{"multi-word, no email", "Two Tokens", "", []string{"two"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aliasKeys(tt.username, tt.email)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// resolveMentionMembers — short-circuit when body has no @
// ---------------------------------------------------------------------------

func TestResolveMentionMembers_SkipsFetchWithoutAtSign(t *testing.T) {
	// Passing nil factory/client would panic if the function tried to fetch.
	// The `@`-less body must short-circuit before any factory access.
	members, err := resolveMentionMembers(nil, nil, "no mentions here")
	assert.NoError(t, err)
	assert.Nil(t, members)
}

// ---------------------------------------------------------------------------
// isWordChar
// ---------------------------------------------------------------------------

func TestIsWordChar(t *testing.T) {
	for c := byte('a'); c <= 'z'; c++ {
		assert.True(t, isWordChar(c))
	}
	for _, c := range []byte{' ', '@', '-', '.', '!'} {
		assert.False(t, isWordChar(c))
	}
}
