package link

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLinksBlock_NoBlock(t *testing.T) {
	desc := "This is a task description with no links block."
	entries, rest := parseLinksBlock(desc)

	assert.Empty(t, entries)
	assert.Equal(t, desc, rest)
}

func TestParseLinksBlock_EmptyDescription(t *testing.T) {
	entries, rest := parseLinksBlock("")

	assert.Empty(t, entries)
	assert.Equal(t, "", rest)
}

func TestParseLinksBlock_WithBlock(t *testing.T) {
	desc := `**GitHub** _(clickup-cli)_
- [owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)
- Branch: ` + "`feat/thing`" + ` in owner/repo

Some task description here.`

	entries, rest := parseLinksBlock(desc)

	assert.Equal(t, []string{
		"[owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)",
		"Branch: `feat/thing` in owner/repo",
	}, entries)
	assert.Equal(t, "Some task description here.", rest)
}

func TestParseLinksBlock_BlockOnly(t *testing.T) {
	desc := `**GitHub** _(clickup-cli)_
- [owner/repo#1 — Title](https://github.com/owner/repo/pull/1)`

	entries, rest := parseLinksBlock(desc)

	assert.Equal(t, []string{
		"[owner/repo#1 — Title](https://github.com/owner/repo/pull/1)",
	}, entries)
	assert.Equal(t, "", rest)
}

func TestParseLinksBlock_DescriptionBeforeAndAfter(t *testing.T) {
	desc := `Before content.

**GitHub** _(clickup-cli)_
- [owner/repo#1 — Title](url)

After content.`

	entries, rest := parseLinksBlock(desc)

	assert.Equal(t, []string{
		"[owner/repo#1 — Title](url)",
	}, entries)
	assert.Equal(t, "Before content.\n\nAfter content.", rest)
}

func TestBuildLinksSection(t *testing.T) {
	entries := []string{
		"[owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)",
		"Branch: `feat/thing` in owner/repo",
	}

	result := buildLinksSection(entries)

	expected := "**GitHub** _(clickup-cli)_\n" +
		"- [owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)\n" +
		"- Branch: `feat/thing` in owner/repo"

	assert.Equal(t, expected, result)
}

func TestBuildLinksSection_SingleEntry(t *testing.T) {
	entries := []string{
		"[owner/repo#1 — Title](url)",
	}

	result := buildLinksSection(entries)

	expected := "**GitHub** _(clickup-cli)_\n- [owner/repo#1 — Title](url)"

	assert.Equal(t, expected, result)
}

func TestUpdateLinksBlock_NewBlockOnEmptyDescription(t *testing.T) {
	entry := linkEntry{
		Prefix: "owner/repo#42",
		Line:   "[owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)",
	}

	result := updateLinksBlock("", entry)

	expected := "**GitHub** _(clickup-cli)_\n- [owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)"

	assert.Equal(t, expected, result)
}

func TestUpdateLinksBlock_NewBlockOnExistingDescription(t *testing.T) {
	entry := linkEntry{
		Prefix: "owner/repo#42",
		Line:   "[owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)",
	}

	result := updateLinksBlock("Existing task description.", entry)

	expected := "**GitHub** _(clickup-cli)_\n" +
		"- [owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)\n\n" +
		"Existing task description."

	assert.Equal(t, expected, result)
}

func TestUpdateLinksBlock_AppendToExistingBlock(t *testing.T) {
	desc := "**GitHub** _(clickup-cli)_\n" +
		"- [owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)\n\n" +
		"Task description."

	entry := linkEntry{
		Prefix: "`feat/thing` in owner/repo",
		Line:   "Branch: `feat/thing` in owner/repo",
	}

	result := updateLinksBlock(desc, entry)

	expected := "**GitHub** _(clickup-cli)_\n" +
		"- [owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)\n" +
		"- Branch: `feat/thing` in owner/repo\n\n" +
		"Task description."

	assert.Equal(t, expected, result)
}

func TestUpdateLinksBlock_DeduplicateByPrefix(t *testing.T) {
	desc := "**GitHub** _(clickup-cli)_\n" +
		"- [owner/repo#42 — Old title](https://github.com/owner/repo/pull/42)\n" +
		"- Branch: `feat/thing` in owner/repo\n\n" +
		"Task description."

	entry := linkEntry{
		Prefix: "owner/repo#42",
		Line:   "[owner/repo#42 — Updated title](https://github.com/owner/repo/pull/42)",
	}

	result := updateLinksBlock(desc, entry)

	expected := "**GitHub** _(clickup-cli)_\n" +
		"- [owner/repo#42 — Updated title](https://github.com/owner/repo/pull/42)\n" +
		"- Branch: `feat/thing` in owner/repo\n\n" +
		"Task description."

	assert.Equal(t, expected, result)
}

func TestUpdateLinksBlock_CrossCuttingCard(t *testing.T) {
	desc := ""

	// Step 1: PR from repo A.
	entry1 := linkEntry{
		Prefix: "triptechtravel/campermate-react-native#33",
		Line:   "[triptechtravel/campermate-react-native#33 — Migrate geozone](https://github.com/triptechtravel/campermate-react-native/pull/33)",
	}
	desc = updateLinksBlock(desc, entry1)

	// Step 2: Branch from repo B.
	entry2 := linkEntry{
		Prefix: "`feat/CU-86d1rn980-geozone-v2` in triptechtravel/cloudflare-worker-functions",
		Line:   "Branch: `feat/CU-86d1rn980-geozone-v2` in triptechtravel/cloudflare-worker-functions",
	}
	desc = updateLinksBlock(desc, entry2)

	// Step 3: PR from repo B.
	entry3 := linkEntry{
		Prefix: "triptechtravel/cloudflare-worker-functions#2",
		Line:   "[triptechtravel/cloudflare-worker-functions#2 — Migrate geozone](https://github.com/triptechtravel/cloudflare-worker-functions/pull/2)",
	}
	desc = updateLinksBlock(desc, entry3)

	assert.Contains(t, desc, "campermate-react-native#33")
	assert.Contains(t, desc, "cloudflare-worker-functions#2")
	assert.Contains(t, desc, "`feat/CU-86d1rn980-geozone-v2`")
}

func TestUpdateLinksBlock_IdempotentRerun(t *testing.T) {
	entry := linkEntry{
		Prefix: "owner/repo#42",
		Line:   "[owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)",
	}

	result1 := updateLinksBlock("Task description.", entry)
	result2 := updateLinksBlock(result1, entry)

	assert.Equal(t, result1, result2)
}

func TestParseLinksBlock_ClickUpStarBullets(t *testing.T) {
	// ClickUp's markdown export uses "*   " bullets instead of "- ".
	desc := "**GitHub** _(clickup-cli)_\n\n*   campermate.com#1109 — SEO\n*   Branch: `feat/seo` in campermate.com\n\nTask description."

	entries, rest := parseLinksBlock(desc)

	assert.Equal(t, []string{
		"campermate.com#1109 — SEO",
		"Branch: `feat/seo` in campermate.com",
	}, entries)
	assert.Equal(t, "Task description.", rest)
}

func TestExtractURL_MarkdownLink(t *testing.T) {
	line := "[owner/repo#42 — Fix bug](https://github.com/owner/repo/pull/42)"
	assert.Equal(t, "https://github.com/owner/repo/pull/42", extractURL(line))
}

func TestExtractURL_PlainParenthesized(t *testing.T) {
	line := "PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)"
	assert.Equal(t, "https://github.com/owner/repo/pull/42", extractURL(line))
}

func TestExtractURL_NoURL(t *testing.T) {
	line := "Branch: `feat/thing` in owner/repo"
	assert.Equal(t, "", extractURL(line))
}
