package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The spec patch pins every millisecond field to clickup.Millis via x-go-type,
// because ClickUp sends those fields as a number on one task and a string on
// the next. This generator resolves the $ref chains the OpenAPI codegen cannot,
// so it has to honour the same instruction — otherwise a field that reaches a
// struct through a $ref is emitted as a rigid scalar and reintroduces issue #27
// on whichever endpoint that struct serves.
func TestSchemaToGoType_HonoursXGoType(t *testing.T) {
	got := schemaToGoType(map[string]any{
		"type":      []any{"integer", "null"},
		"x-go-type": "clickup.Millis",
	})
	assert.Equal(t, "clickup.Millis", got)
}

// A union type is not a Go type. Falling back to `any` keeps the decode working
// for anything the patch has not claimed, rather than guessing one branch of
// the union and failing on the other.
func TestSchemaToGoType_UnionFallsBackToAny(t *testing.T) {
	assert.Equal(t, "any", schemaToGoType(map[string]any{"type": []any{"string", "null"}}))
}

func TestSchemaToGoType_Scalars(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"string", "string"},
		{"integer", "int"},
		{"number", "float64"},
		{"boolean", "bool"},
		{"array", "[]any"},
		{"object", "any"},
		{"", "any"},
	} {
		assert.Equal(t, tt.want, schemaToGoType(map[string]any{"type": tt.in}), "type %q", tt.in)
	}
}

// A field typed by x-go-type needs its package imported, or the fixes file does
// not compile — and it is generated code, so the failure lands on whoever next
// runs `make api-gen`, not on whoever changed the spec.
func TestImportsFor_CollectsXGoTypeImports(t *testing.T) {
	fields := []fieldInfo{
		{Name: "Duration", JSONName: "duration", GoType: "clickup.Millis",
			Import: importSpec{Name: "clickup", Path: "github.com/triptechtravel/clickup-cli/internal/clickup"}},
		{Name: "ID", JSONName: "id", GoType: "string"},
		{Name: "Start", JSONName: "start", GoType: "clickup.Millis",
			Import: importSpec{Name: "clickup", Path: "github.com/triptechtravel/clickup-cli/internal/clickup"}},
	}

	got := importsFor([][]fieldInfo{fields})
	assert.Equal(t, []importSpec{{Name: "clickup", Path: "github.com/triptechtravel/clickup-cli/internal/clickup"}}, got,
		"one import per package, deduplicated")

	assert.Empty(t, importsFor([][]fieldInfo{{{Name: "ID", GoType: "string"}}}),
		"a file with no x-go-type field needs no import block")
}

func TestSchemaToGoType_CapturesImport(t *testing.T) {
	prop := map[string]any{
		"type":      []any{"integer", "null"},
		"x-go-type": "clickup.Millis",
		"x-go-type-import": map[string]any{
			"name": "clickup",
			"path": "github.com/triptechtravel/clickup-cli/internal/clickup",
		},
	}
	assert.Equal(t, importSpec{Name: "clickup", Path: "github.com/triptechtravel/clickup-cli/internal/clickup"},
		importFor(prop))
}
