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
	desc := `--- GitHub Links (clickup-cli) ---
PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)
Branch: feat/thing in owner/repo
--- /GitHub Links ---

Some task description here.`

	entries, rest := parseLinksBlock(desc)

	assert.Equal(t, []string{
		"PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)",
		"Branch: feat/thing in owner/repo",
	}, entries)
	assert.Equal(t, "Some task description here.", rest)
}

func TestParseLinksBlock_BlockOnly(t *testing.T) {
	desc := `--- GitHub Links (clickup-cli) ---
PR: owner/repo#1 - Title (https://github.com/owner/repo/pull/1)
--- /GitHub Links ---`

	entries, rest := parseLinksBlock(desc)

	assert.Equal(t, []string{
		"PR: owner/repo#1 - Title (https://github.com/owner/repo/pull/1)",
	}, entries)
	assert.Equal(t, "", rest)
}

func TestParseLinksBlock_DescriptionBeforeAndAfter(t *testing.T) {
	desc := `Before content.

--- GitHub Links (clickup-cli) ---
PR: owner/repo#1 - Title (url)
--- /GitHub Links ---

After content.`

	entries, rest := parseLinksBlock(desc)

	assert.Equal(t, []string{
		"PR: owner/repo#1 - Title (url)",
	}, entries)
	assert.Equal(t, "Before content.\n\nAfter content.", rest)
}

func TestBuildLinksSection(t *testing.T) {
	entries := []string{
		"PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)",
		"Branch: feat/thing in owner/repo",
	}

	result := buildLinksSection(entries)

	expected := `--- GitHub Links (clickup-cli) ---
PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)
Branch: feat/thing in owner/repo
--- /GitHub Links ---`

	assert.Equal(t, expected, result)
}

func TestBuildLinksSection_SingleEntry(t *testing.T) {
	entries := []string{
		"PR: owner/repo#1 - Title (url)",
	}

	result := buildLinksSection(entries)

	expected := `--- GitHub Links (clickup-cli) ---
PR: owner/repo#1 - Title (url)
--- /GitHub Links ---`

	assert.Equal(t, expected, result)
}

func TestUpdateLinksBlock_NewBlockOnEmptyDescription(t *testing.T) {
	entry := linkEntry{
		Prefix: "PR: owner/repo#42",
		Line:   "PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)",
	}

	result := updateLinksBlock("", entry)

	expected := `--- GitHub Links (clickup-cli) ---
PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)
--- /GitHub Links ---`

	assert.Equal(t, expected, result)
}

func TestUpdateLinksBlock_NewBlockOnExistingDescription(t *testing.T) {
	entry := linkEntry{
		Prefix: "PR: owner/repo#42",
		Line:   "PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)",
	}

	result := updateLinksBlock("Existing task description.", entry)

	expected := `--- GitHub Links (clickup-cli) ---
PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)
--- /GitHub Links ---

Existing task description.`

	assert.Equal(t, expected, result)
}

func TestUpdateLinksBlock_AppendToExistingBlock(t *testing.T) {
	desc := `--- GitHub Links (clickup-cli) ---
PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)
--- /GitHub Links ---

Task description.`

	entry := linkEntry{
		Prefix: "Branch: feat/thing in owner/repo",
		Line:   "Branch: feat/thing in owner/repo",
	}

	result := updateLinksBlock(desc, entry)

	expected := `--- GitHub Links (clickup-cli) ---
PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)
Branch: feat/thing in owner/repo
--- /GitHub Links ---

Task description.`

	assert.Equal(t, expected, result)
}

func TestUpdateLinksBlock_DeduplicateByPrefix(t *testing.T) {
	desc := `--- GitHub Links (clickup-cli) ---
PR: owner/repo#42 - Old title (https://github.com/owner/repo/pull/42)
Branch: feat/thing in owner/repo
--- /GitHub Links ---

Task description.`

	entry := linkEntry{
		Prefix: "PR: owner/repo#42",
		Line:   "PR: owner/repo#42 - Updated title (https://github.com/owner/repo/pull/42)",
	}

	result := updateLinksBlock(desc, entry)

	expected := `--- GitHub Links (clickup-cli) ---
PR: owner/repo#42 - Updated title (https://github.com/owner/repo/pull/42)
Branch: feat/thing in owner/repo
--- /GitHub Links ---

Task description.`

	assert.Equal(t, expected, result)
}

func TestUpdateLinksBlock_CrossCuttingCard(t *testing.T) {
	// Simulate: first PR from repo A, then branch from repo B, then PR from repo B.
	desc := ""

	// Step 1: PR from repo A.
	entry1 := linkEntry{
		Prefix: "PR: triptechtravel/campermate-react-native#33",
		Line:   "PR: triptechtravel/campermate-react-native#33 - Migrate geozone (https://github.com/triptechtravel/campermate-react-native/pull/33)",
	}
	desc = updateLinksBlock(desc, entry1)

	// Step 2: Branch from repo B.
	entry2 := linkEntry{
		Prefix: "Branch: feat/CU-86d1rn980-geozone-v2 in triptechtravel/cloudflare-worker-functions",
		Line:   "Branch: feat/CU-86d1rn980-geozone-v2 in triptechtravel/cloudflare-worker-functions",
	}
	desc = updateLinksBlock(desc, entry2)

	// Step 3: PR from repo B (replaces branch? No, different prefix).
	entry3 := linkEntry{
		Prefix: "PR: triptechtravel/cloudflare-worker-functions#2",
		Line:   "PR: triptechtravel/cloudflare-worker-functions#2 - Migrate geozone (https://github.com/triptechtravel/cloudflare-worker-functions/pull/2)",
	}
	desc = updateLinksBlock(desc, entry3)

	assert.Contains(t, desc, "campermate-react-native#33")
	assert.Contains(t, desc, "cloudflare-worker-functions#2")
	assert.Contains(t, desc, "Branch: feat/CU-86d1rn980-geozone-v2")
}

func TestUpdateLinksBlock_IdempotentRerun(t *testing.T) {
	entry := linkEntry{
		Prefix: "PR: owner/repo#42",
		Line:   "PR: owner/repo#42 - Fix bug (https://github.com/owner/repo/pull/42)",
	}

	// First run.
	result1 := updateLinksBlock("Task description.", entry)

	// Second run with same entry.
	result2 := updateLinksBlock(result1, entry)

	// Should be identical.
	assert.Equal(t, result1, result2)
}
