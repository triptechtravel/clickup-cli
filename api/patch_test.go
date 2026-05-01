package api

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// patchPath returns the absolute path to patch-v2-spec.jq next to this test.
func patchPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "specs", "patch-v2-spec.jq")
}

// runPatch pipes a synthetic spec through `jq -f patch-v2-spec.jq` and
// returns the parsed result. Tests stay light by feeding minimal specs
// that exercise one fix at a time.
func runPatch(t *testing.T, spec map[string]any) map[string]any {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not available")
	}

	in, err := json.Marshal(spec)
	require.NoError(t, err)

	cmd := exec.Command("jq", "-f", patchPath(t))
	cmd.Stdin = bytes.NewReader(in)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("jq failed: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), "patched output: %s", stdout.String())
	return out
}

// dig walks a JSON object via a sequence of keys. Returns nil if any
// segment is missing — callers assert the wanted shape themselves.
func dig(obj any, path ...string) any {
	cur := obj
	for _, k := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[k]
	}
	return cur
}

// ---------------------------------------------------------------------------
// Field-level fixes (walk over `properties`)
// ---------------------------------------------------------------------------

func TestPatch_FixesAssigneesWidening(t *testing.T) {
	spec := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"Task": map[string]any{
					"properties": map[string]any{
						"assignees": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	}
	out := runPatch(t, spec)
	items := dig(out, "components", "schemas", "Task", "properties", "assignees", "items")
	itemsMap, ok := items.(map[string]any)
	require.True(t, ok, "items should be an object")
	assert.Equal(t, "object", itemsMap["type"])
	assert.NotNil(t, itemsMap["properties"], "object items should declare properties")
}

func TestPatch_FixesTimeSpentToInteger(t *testing.T) {
	spec := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"Task": map[string]any{
					"properties": map[string]any{
						"time_spent": map[string]any{"type": []any{"string", "null"}},
					},
				},
			},
		},
	}
	out := runPatch(t, spec)
	ts := dig(out, "components", "schemas", "Task", "properties", "time_spent")
	tsMap, ok := ts.(map[string]any)
	require.True(t, ok)
	typeField, _ := tsMap["type"].([]any)
	assert.Contains(t, typeField, "integer", "time_spent should be widened to integer")
}

func TestPatch_FixesTagsToObjectItems(t *testing.T) {
	spec := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"Task": map[string]any{
					"properties": map[string]any{
						"tags": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	}
	out := runPatch(t, spec)
	items := dig(out, "components", "schemas", "Task", "properties", "tags", "items").(map[string]any)
	assert.Equal(t, "object", items["type"])
	assert.Contains(t, items["properties"], "name")
}

// ---------------------------------------------------------------------------
// Comment request bodies — should gain `comment` and `markdown_text`,
// and lose `comment_text`/`assignee`/`resolved` from `required`.
// ---------------------------------------------------------------------------

func TestPatch_AddsCommentAndMarkdownTextToCreateRequest(t *testing.T) {
	spec := map[string]any{
		"paths": map[string]any{
			"/v2/task/{task_id}/comment": map[string]any{
				"post": map[string]any{
					"requestBody": map[string]any{
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"required": []any{"comment_text", "notify_all"},
									"properties": map[string]any{
										"comment_text": map[string]any{"type": "string"},
										"notify_all":   map[string]any{"type": "boolean"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	out := runPatch(t, spec)
	schema := dig(out,
		"paths", "/v2/task/{task_id}/comment", "post",
		"requestBody", "content", "application/json", "schema").(map[string]any)

	props := schema["properties"].(map[string]any)
	assert.Contains(t, props, "comment", "request schema should gain `comment`")
	assert.Contains(t, props, "markdown_text", "request schema should gain `markdown_text`")

	required, _ := schema["required"].([]any)
	for _, r := range required {
		assert.NotEqual(t, "comment_text", r, "comment_text should not be required after patch")
	}
}

func TestPatch_RelaxesUpdateCommentRequired(t *testing.T) {
	spec := map[string]any{
		"paths": map[string]any{
			"/v2/comment/{comment_id}": map[string]any{
				"put": map[string]any{
					"requestBody": map[string]any{
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"required": []any{"comment_text", "assignee", "resolved"},
									"properties": map[string]any{
										"comment_text": map[string]any{"type": "string"},
										"assignee":     map[string]any{"type": "integer"},
										"resolved":     map[string]any{"type": "boolean"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	out := runPatch(t, spec)
	schema := dig(out,
		"paths", "/v2/comment/{comment_id}", "put",
		"requestBody", "content", "application/json", "schema").(map[string]any)

	required, ok := schema["required"].([]any)
	if ok {
		for _, r := range required {
			s, _ := r.(string)
			assert.NotContains(t, []string{"comment_text", "assignee", "resolved"}, s,
				"%s should be relaxed from required for partial PUT updates", s)
		}
	}
}

// ---------------------------------------------------------------------------
// Comment response bodies — id should be widened to integer to match what
// ClickUp actually returns from the create-comment endpoints.
// ---------------------------------------------------------------------------

func TestPatch_FixesCreateCommentResponseIDToInteger(t *testing.T) {
	spec := map[string]any{
		"paths": map[string]any{
			"/v2/task/{task_id}/comment": map[string]any{
				"post": map[string]any{
					"responses": map[string]any{
						"200": map[string]any{
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"properties": map[string]any{
											"id":      map[string]any{"type": "string"},
											"hist_id": map[string]any{"type": "string"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	out := runPatch(t, spec)
	idSchema := dig(out,
		"paths", "/v2/task/{task_id}/comment", "post",
		"responses", "200", "content", "application/json", "schema",
		"properties", "id").(map[string]any)
	assert.Equal(t, "integer", idSchema["type"], "id should be widened to integer to match API behaviour")

	histSchema := dig(out,
		"paths", "/v2/task/{task_id}/comment", "post",
		"responses", "200", "content", "application/json", "schema",
		"properties", "hist_id").(map[string]any)
	assert.Equal(t, "string", histSchema["type"], "hist_id should remain a string")
}

func TestPatch_FixesListCommentResponseIDToInteger(t *testing.T) {
	spec := map[string]any{
		"paths": map[string]any{
			"/v2/list/{list_id}/comment": map[string]any{
				"post": map[string]any{
					"responses": map[string]any{
						"200": map[string]any{
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"properties": map[string]any{
											"id": map[string]any{"type": "string"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	out := runPatch(t, spec)
	idSchema := dig(out,
		"paths", "/v2/list/{list_id}/comment", "post",
		"responses", "200", "content", "application/json", "schema",
		"properties", "id").(map[string]any)
	assert.Equal(t, "integer", idSchema["type"])
}

// ---------------------------------------------------------------------------
// Patch must be a no-op for already-fixed input (idempotent).
// ---------------------------------------------------------------------------

func TestPatch_IdempotentOnAlreadyPatchedInput(t *testing.T) {
	spec := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"Task": map[string]any{
					"properties": map[string]any{
						"assignees": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"id": map[string]any{"type": "integer"},
								},
							},
						},
						"time_spent": map[string]any{"type": []any{"integer", "null"}},
					},
				},
			},
		},
	}
	first := runPatch(t, spec)
	second := runPatch(t, first)

	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	assert.JSONEq(t, string(firstJSON), string(secondJSON),
		"applying the patch twice should be identical to applying once")
}

// ---------------------------------------------------------------------------
// The patched real spec on disk: assert the contract our generated wrappers
// rely on. Skipped in CI environments without the spec materialised.
// ---------------------------------------------------------------------------

func TestPatchedSpec_RealFile_ContractMatchesGeneratedWrappers(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(filepath.Dir(thisFile), "specs", "clickup-v2.json")

	cmd := exec.Command("jq", "-r",
		`.paths."/v2/task/{task_id}/comment".post.requestBody.content."application/json".schema.properties.comment.type,
		 .paths."/v2/task/{task_id}/comment".post.requestBody.content."application/json".schema.properties.markdown_text.type,
		 .paths."/v2/task/{task_id}/comment".post.responses."200".content."application/json".schema.properties.id.type`,
		specPath)
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("real spec not present (run `make api/specs/clickup-v2.json` to materialise): %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	require.Len(t, lines, 3, "expected 3 jq outputs, got: %q", string(out))
	assert.Equal(t, "array", lines[0], "comment field should be an array of blocks")
	assert.Equal(t, "string", lines[1], "markdown_text should be a string")
	assert.Equal(t, "integer", lines[2], "create-comment response id should be integer")
}
