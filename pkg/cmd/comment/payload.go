package comment

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// commentBlock is one entry in a ClickUp structured comment payload, encoded
// as a Quill-style delta. Plain text spans use Text + Attributes; @mentions
// use Type="tag" + User.
type commentBlock struct {
	Text       string                 `json:"text,omitempty"`
	Type       string                 `json:"type,omitempty"`
	User       *mentionUser           `json:"user,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type mentionUser struct {
	ID int `json:"id"`
}

type workspaceMember struct {
	Username string
	ID       int
}

// buildBlocks converts a markdown body into Quill-delta blocks and resolves
// @mentions. The bool return reports whether the structured form is needed —
// either because the body contains formatting or because mentions resolved.
// Callers send the structured `comment` field when true and fall back to the
// plain `comment_text` field otherwise.
func buildBlocks(body string, members map[string]workspaceMember) ([]commentBlock, []string, bool) {
	blocks := markdownToBlocks(body)
	blocks, resolved := resolveMentionsInBlocks(blocks, members)
	useBlocks := len(resolved) > 0 || hasFormatting(blocks)
	return blocks, resolved, useBlocks
}

// resolveMentionMembers fetches workspace members only when the body actually
// contains an `@`. Other callers (commands without mention support) can pass
// nil to buildBlocks. Errors are returned so callers can surface a warning
// rather than silently dropping mention resolution.
func resolveMentionMembers(f *cmdutil.Factory, client *api.Client, body string) (map[string]workspaceMember, error) {
	if !strings.Contains(body, "@") {
		return nil, nil
	}
	return fetchWorkspaceMembers(f, client)
}

// toCreateBlocks adapts internal commentBlocks to the typed slice the
// generated CreateTaskComment wrapper expects.
func toCreateBlocks(blocks []commentBlock) []clickupv2.PostV2TaskTaskIDCommentRequestJSON2 {
	out := make([]clickupv2.PostV2TaskTaskIDCommentRequestJSON2, 0, len(blocks))
	for _, b := range blocks {
		var item clickupv2.PostV2TaskTaskIDCommentRequestJSON2
		if b.Text != "" {
			t := b.Text
			item.Text = &t
		}
		if b.Type != "" {
			tp := b.Type
			item.Type = &tp
		}
		if b.User != nil {
			id := b.User.ID
			item.User = &clickupv2.PostV2TaskTaskIDCommentRequestJSON3{ID: &id}
		}
		if len(b.Attributes) > 0 {
			item.Attributes = b.Attributes
		}
		out = append(out, item)
	}
	return out
}

// toUpdateBlocks is the same conversion for the UpdateComment endpoint, which
// has its own typed slice.
func toUpdateBlocks(blocks []commentBlock) []clickupv2.PutV2CommentCommentIDRequestJSON2 {
	out := make([]clickupv2.PutV2CommentCommentIDRequestJSON2, 0, len(blocks))
	for _, b := range blocks {
		var item clickupv2.PutV2CommentCommentIDRequestJSON2
		if b.Text != "" {
			t := b.Text
			item.Text = &t
		}
		if b.Type != "" {
			tp := b.Type
			item.Type = &tp
		}
		if b.User != nil {
			id := b.User.ID
			item.User = &clickupv2.PutV2CommentCommentIDRequestJSON3{ID: &id}
		}
		if len(b.Attributes) > 0 {
			item.Attributes = b.Attributes
		}
		out = append(out, item)
	}
	return out
}

// toReplyBlocks adapts internal commentBlocks to the typed slice the
// generated CreateThreadedComment wrapper expects.
func toReplyBlocks(blocks []commentBlock) []clickupv2.PostV2CommentCommentIDReplyRequestJSON2 {
	out := make([]clickupv2.PostV2CommentCommentIDReplyRequestJSON2, 0, len(blocks))
	for _, b := range blocks {
		var item clickupv2.PostV2CommentCommentIDReplyRequestJSON2
		if b.Text != "" {
			t := b.Text
			item.Text = &t
		}
		if b.Type != "" {
			tp := b.Type
			item.Type = &tp
		}
		if b.User != nil {
			id := b.User.ID
			item.User = &clickupv2.PostV2CommentCommentIDReplyRequestJSON3{ID: &id}
		}
		if len(b.Attributes) > 0 {
			item.Attributes = b.Attributes
		}
		out = append(out, item)
	}
	return out
}

// hasFormatting reports whether any block carries inline or block-level
// formatting (or is a mention tag) — i.e. whether sending as `comment_text`
// would lose information.
func hasFormatting(blocks []commentBlock) bool {
	for _, b := range blocks {
		if b.Type == "tag" || len(b.Attributes) > 0 {
			return true
		}
	}
	return false
}

// memberCache holds workspace members for the duration of a CLI process,
// keyed by workspace ID. Chained operations (`comment add` followed by
// another `comment add` in a loop, or a single command that resolves
// multiple mentions) reuse the cached lookup instead of refetching.
var (
	memberCacheMu sync.Mutex
	memberCache   = map[string]map[string]workspaceMember{}
)

// fetchWorkspaceMembers returns a lookup of mention key → workspace member.
// Members are keyed by lowercased full username, and additionally by their
// first-name token and email local-part when those are unambiguous within
// the workspace — so `@first` resolves to a member named "First Last" when
// they're the only First in the workspace. Results are cached per-workspace
// for the lifetime of the process.
func fetchWorkspaceMembers(f *cmdutil.Factory, client *api.Client) (map[string]workspaceMember, error) {
	cfg, err := f.Config()
	if err != nil {
		return nil, err
	}
	if cfg.Workspace == "" {
		return nil, fmt.Errorf("no workspace configured")
	}

	memberCacheMu.Lock()
	if cached, ok := memberCache[cfg.Workspace]; ok {
		memberCacheMu.Unlock()
		return cached, nil
	}
	memberCacheMu.Unlock()

	ctx := context.Background()
	teams, err := apiv2.GetTeamsLocal(ctx, client)
	if err != nil {
		return nil, err
	}

	members := make(map[string]workspaceMember)
	aliasCount := map[string]int{}

	for _, team := range teams {
		if team.ID != cfg.Workspace {
			continue
		}
		// First pass: register full-username keys and count per-user dedup'd
		// aliases. Counting dedup'd aliases is what makes `@first` work when a
		// member's first-name token equals their email local-part — without
		// dedup the same key is counted twice for one user and (incorrectly)
		// flagged as ambiguous.
		for _, m := range team.Members {
			full := strings.ToLower(m.User.Username)
			members[full] = workspaceMember{Username: m.User.Username, ID: m.User.ID}
			for _, k := range aliasKeys(m.User.Username, m.User.Email) {
				aliasCount[k]++
			}
		}
		// Second pass: bind aliases that are unambiguous (count == 1) and
		// don't shadow an existing full-username key.
		for _, m := range team.Members {
			wm := workspaceMember{Username: m.User.Username, ID: m.User.ID}
			for _, k := range aliasKeys(m.User.Username, m.User.Email) {
				if aliasCount[k] != 1 {
					continue
				}
				if _, taken := members[k]; taken {
					continue
				}
				members[k] = wm
			}
		}
		break
	}

	memberCacheMu.Lock()
	memberCache[cfg.Workspace] = members
	memberCacheMu.Unlock()

	return members, nil
}

// aliasKeys returns lowercase mention shortcuts for a member: the first-name
// token of a multi-word username, the dotted prefix of an email local-part
// (e.g. "first.last@…" → "first"), and the full email local-part. Returned
// keys are deduplicated so a user whose first-name token matches their email
// local-part is counted once when checking for ambiguity.
func aliasKeys(username, email string) []string {
	seen := map[string]bool{}
	var keys []string
	add := func(k string) {
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		keys = append(keys, k)
	}

	lower := strings.ToLower(username)
	if i := strings.IndexAny(lower, " \t"); i > 0 {
		add(lower[:i])
	}
	if at := strings.IndexByte(email, '@'); at > 0 {
		local := strings.ToLower(email[:at])
		if dot := strings.IndexByte(local, '.'); dot > 0 {
			add(local[:dot])
		}
		add(local)
	}
	return keys
}

// isWordChar reports whether a byte is part of a username token — used to
// reject e.g. `@al` matching inside `@alex`.
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// resolveMentionsInBlocks scans every block's text for @username runs and
// splits matching blocks into [pre-text, tag, post-text] triples. Inline
// attributes from the original block are preserved on the surrounding text.
// Returns the rewritten blocks and the display names of resolved mentions.
func resolveMentionsInBlocks(blocks []commentBlock, members map[string]workspaceMember) ([]commentBlock, []string) {
	if len(members) == 0 {
		return blocks, nil
	}

	type sortedMember struct {
		lower string
		m     workspaceMember
	}
	var sorted []sortedMember
	for k, v := range members {
		sorted = append(sorted, sortedMember{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].lower) > len(sorted[j].lower)
	})

	var out []commentBlock
	var resolved []string

	for _, b := range blocks {
		if b.Type == "tag" || b.Text == "" || !strings.Contains(b.Text, "@") {
			out = append(out, b)
			continue
		}
		body := b.Text
		bodyLower := strings.ToLower(body)
		pos := 0
		for i := 0; i < len(body); i++ {
			if body[i] != '@' || i+1 >= len(body) {
				continue
			}
			afterAt := bodyLower[i+1:]
			for _, sm := range sorted {
				if !strings.HasPrefix(afterAt, sm.lower) {
					continue
				}
				endPos := i + 1 + len(sm.lower)
				if endPos < len(body) && isWordChar(body[endPos]) {
					continue
				}
				if i > pos {
					nb := b
					nb.Text = body[pos:i]
					out = append(out, nb)
				}
				out = append(out, commentBlock{Type: "tag", User: &mentionUser{ID: sm.m.ID}})
				resolved = append(resolved, sm.m.Username)
				pos = endPos
				i = endPos - 1
				break
			}
		}
		if pos < len(body) {
			nb := b
			nb.Text = body[pos:]
			out = append(out, nb)
		}
	}

	return out, resolved
}

// markdownToBlocks parses a markdown body and emits Quill-delta blocks. Block
// attributes (header, list, code-block, blockquote) ride on the trailing \n
// for that line; inline attributes (bold, italic, code, link) ride on the
// text span itself. Unknown nodes fall through to plain text, so partially
// understood markdown still posts intelligibly.
func markdownToBlocks(body string) []commentBlock {
	src := []byte(body)
	doc := goldmark.New().Parser().Parse(text.NewReader(src))
	w := &deltaWriter{src: src}
	w.walkBlocks(doc)
	return w.blocks
}

type deltaWriter struct {
	src    []byte
	blocks []commentBlock
}

func (w *deltaWriter) walkBlocks(parent ast.Node) {
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		w.block(n)
	}
}

func (w *deltaWriter) block(n ast.Node) {
	switch v := n.(type) {
	case *ast.Heading:
		w.inlines(v, nil)
		w.append("\n", map[string]interface{}{"header": v.Level})
	case *ast.Paragraph:
		w.inlines(v, nil)
		w.append("\n", nil)
	case *ast.List:
		w.list(v, 0)
	case *ast.FencedCodeBlock:
		w.emitCodeLines(v.Lines())
	case *ast.CodeBlock:
		w.emitCodeLines(v.Lines())
	case *ast.Blockquote:
		for c := v.FirstChild(); c != nil; c = c.NextSibling() {
			if p, ok := c.(*ast.Paragraph); ok {
				w.inlines(p, nil)
				w.append("\n", map[string]interface{}{"blockquote": true})
				continue
			}
			w.block(c)
		}
	case *ast.ThematicBreak:
		w.append("---", nil)
		w.append("\n", nil)
	default:
		w.walkBlocks(v)
	}
}

func (w *deltaWriter) inlines(parent ast.Node, attrs map[string]interface{}) {
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		switch cc := c.(type) {
		case *ast.Text:
			s := string(cc.Segment.Value(w.src))
			w.append(s, attrs)
			if cc.HardLineBreak() {
				w.append("\n", attrs)
			} else if cc.SoftLineBreak() {
				w.append(" ", attrs)
			}
		case *ast.Emphasis:
			key := "italic"
			if cc.Level == 2 {
				key = "bold"
			}
			w.inlines(cc, mergeAttr(attrs, key, true))
		case *ast.CodeSpan:
			var sb strings.Builder
			for cs := cc.FirstChild(); cs != nil; cs = cs.NextSibling() {
				if tn, ok := cs.(*ast.Text); ok {
					sb.Write(tn.Segment.Value(w.src))
				}
			}
			w.append(sb.String(), mergeAttr(attrs, "code", true))
		case *ast.Link:
			w.inlines(cc, mergeAttr(attrs, "link", string(cc.Destination)))
		case *ast.AutoLink:
			url := string(cc.URL(w.src))
			w.append(url, mergeAttr(attrs, "link", url))
		case *ast.RawHTML:
			for i := 0; i < cc.Segments.Len(); i++ {
				seg := cc.Segments.At(i)
				w.append(string(seg.Value(w.src)), attrs)
			}
		default:
			w.inlines(cc, attrs)
		}
	}
}

// list walks a list (and any lists nested inside its items), emitting one
// list-typed newline per item. Nested lists carry an `indent` attribute so
// Quill renders them as sub-items rather than collapsing into the parent.
func (w *deltaWriter) list(v *ast.List, indent int) {
	listType := "bullet"
	if v.IsOrdered() {
		listType = "ordered"
	}
	for li := v.FirstChild(); li != nil; li = li.NextSibling() {
		// Split each item's children into the first textual block (the item's
		// own line) and everything else (nested lists, additional paragraphs).
		// The item's terminating newline must come *after* its text but
		// *before* any nested list, otherwise the nested list's last newline
		// ends up carrying the parent's list type.
		var firstText ast.Node
		var rest []ast.Node
		for c := li.FirstChild(); c != nil; c = c.NextSibling() {
			switch c.(type) {
			case *ast.TextBlock, *ast.Paragraph:
				if firstText == nil {
					firstText = c
					continue
				}
			}
			rest = append(rest, c)
		}
		if firstText != nil {
			w.inlines(firstText, nil)
		}
		attrs := map[string]interface{}{"list": listType}
		if indent > 0 {
			attrs["indent"] = indent
		}
		w.append("\n", attrs)
		for _, c := range rest {
			if nested, ok := c.(*ast.List); ok {
				w.list(nested, indent+1)
				continue
			}
			w.block(c)
		}
	}
}

func (w *deltaWriter) emitCodeLines(segs *text.Segments) {
	for i := 0; i < segs.Len(); i++ {
		seg := segs.At(i)
		line := strings.TrimSuffix(string(seg.Value(w.src)), "\n")
		w.append(line, nil)
		w.append("\n", map[string]interface{}{"code-block": true})
	}
}

func (w *deltaWriter) append(s string, attrs map[string]interface{}) {
	if s == "" {
		return
	}
	b := commentBlock{Text: s}
	if len(attrs) > 0 {
		b.Attributes = copyAttrs(attrs)
	}
	w.blocks = append(w.blocks, b)
}

func mergeAttr(base map[string]interface{}, key string, val interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(base)+1)
	for k, v := range base {
		out[k] = v
	}
	out[key] = val
	return out
}

func copyAttrs(a map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	return out
}
