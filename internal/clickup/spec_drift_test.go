package clickup_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
)

// Drift between the hand-written types and the generated ones is how a field
// that exists in the API becomes a feature that does not exist in the CLI.
//
// Two real cases:
//   - `archived` was in the spec and in the generated request type, but absent
//     from clickup.TaskUpdateRequest, so `task archive` was unbuildable and
//     archiving 60 tasks had to be done with curl.
//   - `locations` was absent from clickup.Task, so the CLI could not see that a
//     list view was hiding multi-list tasks — and reported the list clean twice.
//
// Porting every command to generated types (docs/api-layer-plan.md Phase 4) is
// the eventual fix, but it is a large change and it is not what stops the next
// field going missing. This test does: it compares json tags field-for-field
// and fails when the spec has something the hand-written type does not.
//
// To resolve a failure, either add the field, or add it to the allowlist below
// with a reason. Never delete the assertion.
func TestSpecDrift_HandWrittenTypesTrackTheSpec(t *testing.T) {
	cases := []struct {
		name      string
		handWrit  any
		generated any
		// allowed maps a json field to why it is deliberately not modelled.
		allowed map[string]string
	}{
		{
			name:      "TaskUpdateRequest vs UpdateTaskJSONRequest",
			handWrit:  clickup.TaskUpdateRequest{},
			generated: clickupv2.UpdateTaskJSONRequest{},
			allowed: map[string]string{
				"group_assignees": "user groups are unsupported by the CLI; no command sets them",
			},
		},
		{
			name:      "Task vs GetTaskJSONResponse",
			handWrit:  clickup.Task{},
			generated: clickupv2.GetTaskJSONResponse{},
			allowed: map[string]string{
				"group_assignees": "user groups are unsupported by the CLI",
				"sharing":         "share settings are not surfaced by any command",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			have := jsonFields(tc.handWrit)
			want := jsonFields(tc.generated)

			var missing []string
			for f := range want {
				if have[f] {
					continue
				}
				if _, ok := tc.allowed[f]; ok {
					continue
				}
				missing = append(missing, f)
			}
			sort.Strings(missing)

			assert.Empty(t, missing,
				"%s is missing spec fields.\n"+
					"Add them to the hand-written type, or allowlist them with a reason.\n"+
					"Pointer types for anything clearable — a bare bool with omitempty\n"+
					"serialises to nothing and silently no-ops (see --unarchive).",
				reflect.TypeOf(tc.handWrit).Name())
		})
	}
}

// jsonFields returns the set of json tag names on a struct, ignoring options
// like omitempty and skipping fields excluded from serialisation.
func jsonFields(v any) map[string]bool {
	out := map[string]bool{}
	rt := reflect.TypeOf(v)
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name != "" {
			out[name] = true
		}
	}
	return out
}
