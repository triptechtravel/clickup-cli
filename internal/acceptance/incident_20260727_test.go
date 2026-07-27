// Package acceptance encodes, as executable tests, the gaps found on
// 2026-07-27 while cleaning out a ClickUp list.
//
// Every case below is something that had to be done by hand — with raw curl, a
// keychain excavation, or a Python scan across every list in the workspace —
// because the CLI either could not do it or reported success without doing it.
//
// Each test states:
//
//	MANUAL: what was actually run on the day
//	WANT:   the CLI invocation that should replace it
//
// Unimplemented phases are skipped. `go test -run Incident -v ./internal/acceptance`
// prints the backlog; deleting a t.Skip is the acceptance criterion for that
// phase. Do not delete a skip without making the assertion below it pass.
package acceptance

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

// ---------------------------------------------------------------------------
// Phase 1 — stop lying
// ---------------------------------------------------------------------------

// MANUAL: `clickup task archive 4n6u4x9` printed the task help and exited 0.
// Nothing was archived and nothing said so, which is how the mistake survived.
//
// WANT: unknown verbs fail loudly and name themselves.
func TestIncident_UnknownSubcommandExitsNonZero(t *testing.T) {
	for _, group := range []string{"task", "list", "folder", "space", "doc", "comment"} {
		t.Run(group, func(t *testing.T) {
			tf := testutil.NewTestFactory(t)
			_, _, err := runCLI(t, tf, group, "bogusverb")

			require.Error(t, err, "unknown subcommand must not exit 0")
			assert.Contains(t, err.Error(), "bogusverb", "error should name the bad verb")
		})
	}
}

// A group command invoked bare should still print help and succeed — that is
// the discoverable path, and NoArgs must not break it.
func TestIncident_BareGroupCommandStillShowsHelp(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	out, _, err := runCLI(t, tf, "task")

	require.NoError(t, err, "bare group command should still show help")
	assert.Contains(t, out, "Available Commands")
}

// MANUAL: nothing — this was never noticed, which is the problem. ensure-gen
// only checks api/clickupv3/client.gen.go, so a missing or stale v2 client
// (where every task operation lives) is invisible.
func TestIncident_EnsureGenChecksBothClients(t *testing.T) {
	// Scoped to the ensure-gen recipe: the v2 path appears elsewhere in the
	// Makefile (the api-gen recipe), so a whole-file match passes for the
	// wrong reason.
	recipe := makefileRecipe(t, "ensure-gen")
	assert.Contains(t, recipe, "api/clickupv2/client.gen.go",
		"ensure-gen must guard the v2 client, not just v3")
	assert.Contains(t, recipe, "api/clickupv3/client.gen.go")
}

// MANUAL: the spec is fetched unpinned from developer.clickup.com at build
// time. Generated code is deliberately not committed, so the spec IS the source
// of truth — and two clones a week apart can silently differ.
func TestIncident_SpecIsPinnedByChecksum(t *testing.T) {
	makefile := readRepoFile(t, "Makefile")
	assert.Contains(t, makefile, "SPEC_V2_SHA")
	assert.Contains(t, makefile, "SPEC_V3_SHA")
	assert.Contains(t, makefile, "api-update", "need a deliberate re-pin target")
}

// MANUAL: every archive call was
//
//	TOKEN=$(security find-generic-password -s clickup-cli -a api_token -w \
//	  | python3 -c 'strip the go-keyring-base64: prefix and decode')
//	curl -X PUT -H "Authorization: $TOKEN" -d '{"archived":true}' \
//	  https://api.clickup.com/api/v2/task/$id
//
// ...repeated 60 times in a shell loop. A passthrough makes this a one-liner,
// and makes every future gap an inconvenience rather than a blocker.
func TestIncident_GenericAPIPassthrough(t *testing.T) {
	t.Run("put with typed fields", func(t *testing.T) {
		tf := testutil.NewTestFactory(t)
		var gotMethod string
		var gotBody map[string]any

		tf.HandleFunc("task/4n6u4x9", func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &gotBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"4n6u4x9","archived":true}`)
		})

		out, _, err := runCLI(t, tf, "api", "-X", "PUT", "task/4n6u4x9", "-f", "archived=true")

		require.NoError(t, err)
		assert.Equal(t, http.MethodPut, gotMethod)
		assert.Equal(t, true, gotBody["archived"],
			`-f must coerce true/false to JSON booleans, not send the string "true"`)
		assert.Contains(t, out, "4n6u4x9", "response body goes to stdout")
	})

	t.Run("get defaults to GET and prints json", func(t *testing.T) {
		tf := testutil.NewTestFactory(t)
		tf.Handle(http.MethodGet, "list/216668663/task", 200,
			`{"tasks":[{"id":"qa1","name":"[QA] smoke"}]}`)

		out, _, err := runCLI(t, tf, "api", "list/216668663/task")

		require.NoError(t, err)
		assert.Contains(t, out, "[QA] smoke")
	})

	t.Run("raw string fields are not coerced", func(t *testing.T) {
		tf := testutil.NewTestFactory(t)
		var gotBody map[string]any
		tf.HandleFunc("task/abc", func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &gotBody)
			_, _ = io.WriteString(w, `{}`)
		})

		_, _, err := runCLI(t, tf, "api", "-X", "PUT", "task/abc", "--raw-field", "name=true")

		require.NoError(t, err)
		assert.Equal(t, "true", gotBody["name"], "--raw-field must preserve the string")
	})

	t.Run("api errors surface non-zero with the body", func(t *testing.T) {
		tf := testutil.NewTestFactory(t)
		tf.Handle(http.MethodGet, "task/missing", 404, `{"err":"Task not found","ECODE":"ITEM_013"}`)

		_, _, err := runCLI(t, tf, "api", "task/missing")

		require.Error(t, err, "HTTP errors must not exit 0")
		assert.Contains(t, err.Error(), "ITEM_013", "surface the API error code")
	})

	t.Run("jq filters the response", func(t *testing.T) {
		tf := testutil.NewTestFactory(t)
		tf.Handle(http.MethodGet, "list/1/task", 200,
			`{"tasks":[{"name":"one"},{"name":"two"}]}`)

		out, _, err := runCLI(t, tf, "api", "list/1/task", "--jq", ".tasks[].name")

		require.NoError(t, err)
		assert.Contains(t, out, "one")
		assert.Contains(t, out, "two")
	})
}

// ---------------------------------------------------------------------------
// Phase 2 — make membership visible
//
// The expensive failure. `clickup task list --list-id 216668663` reported 2
// tasks; the UI showed 8. The 6 extra were multi-list: homed in Bugs and
// Product/Design, linked into Apps. The CLI has no Locations field, so it could
// not see them and could not say it was not showing them. I reported the list
// clean, twice, and was wrong both times.
//
// MANUAL: a Python scan over every list in every space, fetching each task and
// checking locations[].id for the target list — ~400 requests to answer "what
// is in this list".
// ---------------------------------------------------------------------------

func TestIncident_TaskViewShowsLinkedLists(t *testing.T) {
	

	tf := testutil.NewTestFactory(t)
	tf.Handle(http.MethodGet, "task/4n6u4xw", 200, `{
		"id":"4n6u4xw","name":"iOS Admob placements not showing",
		"status":{"status":"done"},
		"list":{"id":"216686969","name":"Bugs"},
		"locations":[{"id":"216668663","name":"Apps"}]
	}`)

	out, _, err := runCLI(t, tf, "task", "view", "4n6u4xw")

	require.NoError(t, err)
	assert.Contains(t, out, "Bugs", "home list")
	assert.Contains(t, out, "Apps", "linked list must be visible, not silently dropped")
}

// --linked has to scan the workspace: ClickUp has no endpoint that returns
// tasks linked into a list (see api/GO_CLICKUP_GAPS.md). The test asserts the
// scan finds a foreign-homed task and that the cost is disclosed.
func TestIncident_TaskListLinkedFlag(t *testing.T) {
	tf := testutil.NewTestFactory(t)

	// Home-list query returns only the QA task, as the real API does.
	tf.Handle(http.MethodGet, "list/216668663/task", 200,
		`{"tasks":[{"id":"qa1","name":"[QA] smoke","status":{"status":"open"},"list":{"id":"216668663","name":"Apps"}}]}`)

	// Workspace shape for the scan.
	tf.Handle(http.MethodGet, "team", 200, `{"teams":[{"id":"12345","name":"TripTech"}]}`)
	tf.Handle(http.MethodGet, "team/12345/space", 200, `{"spaces":[{"id":"66607490","name":"Development"}]}`)
	tf.Handle(http.MethodGet, "space/66607490/list", 200,
		`{"lists":[{"id":"216668663","name":"Apps"},{"id":"216686969","name":"Bugs"}]}`)
	tf.Handle(http.MethodGet, "space/66607490/folder", 200, `{"folders":[]}`)

	// The Bugs list holds a task linked into Apps.
	tf.Handle(http.MethodGet, "list/216686969/task", 200, `{"tasks":[{
		"id":"4n6u4xw","name":"iOS Admob placements not showing","status":{"status":"done"},
		"list":{"id":"216686969","name":"Bugs"},
		"locations":[{"id":"216668663","name":"Apps"}]}]}`)

	out, errOut, err := runCLI(t, tf, "task", "list", "--list-id", "216668663", "--linked")

	require.NoError(t, err)
	assert.Contains(t, out, "iOS Admob placements", "linked tasks must appear under --linked")
	assert.Contains(t, out, "[QA] smoke", "home-list tasks must still appear")
	assert.Regexp(t, `(?i)scan`, errOut, "the workspace scan cost must be disclosed")
}

// The general fix, and the one that actually matters: a filtered view must
// never read as a complete one. This is what turned a missing feature into a
// wrong answer.
//
// A count of what was hidden is not obtainable without spending the requests
// the user did not ask for, so the honest form is a named caveat plus the flag
// that lifts it.
func TestIncident_FilteredListReportsWhatItOmitted(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle(http.MethodGet, "list/216668663/task", 200,
		`{"tasks":[{"id":"qa1","name":"[QA] smoke","status":{"status":"open"},"list":{"id":"216668663","name":"Apps"}}]}`)

	out, _, err := runCLI(t, tf, "task", "list", "--list-id", "216668663")

	require.NoError(t, err)
	assert.Regexp(t, `(?i)linked`, out,
		"must disclose that tasks linked in from other lists are not shown")
	assert.Contains(t, out, "--linked", "must name the flag that would show them")
	assert.Regexp(t, `(?i)subtask`, out,
		"must disclose that subtasks are excluded by default")
	assert.Contains(t, out, "--include-subtasks")
}

// The caveat must disappear once the flags are supplied, or it becomes noise
// that people learn to ignore.
func TestIncident_NoCaveatWhenNothingIsHidden(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle(http.MethodGet, "list/216668663/task", 200,
		`{"tasks":[{"id":"qa1","name":"[QA] smoke","status":{"status":"open"},"list":{"id":"216668663","name":"Apps"}}]}`)
	tf.Handle(http.MethodGet, "team", 200, `{"teams":[{"id":"12345","name":"TripTech"}]}`)
	tf.Handle(http.MethodGet, "team/12345/space", 200, `{"spaces":[]}`)

	out, _, err := runCLI(t, tf, "task", "list", "--list-id", "216668663",
		"--linked", "--include-subtasks")

	require.NoError(t, err)
	assert.NotRegexp(t, `(?i)not shown`, out,
		"no caveat when nothing is being withheld")
}

// ---------------------------------------------------------------------------
// Phase 3 — archive
// ---------------------------------------------------------------------------

func TestIncident_TaskArchiveAndUnarchive(t *testing.T) {
	t.Skip("Phase 3.7 — clickup task archive/unarchive <ids...>")

	tf := testutil.NewTestFactory(t)
	var bodies []map[string]any
	for _, id := range []string{"4n6u4xw", "4n6u4y9", "4n6u4yb"} {
		tf.HandleFunc("task/"+id, func(w http.ResponseWriter, r *http.Request) {
			var b map[string]any
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &b)
			bodies = append(bodies, b)
			_, _ = io.WriteString(w, `{"id":"x"}`)
		})
	}

	_, _, err := runCLI(t, tf, "task", "archive", "4n6u4xw", "4n6u4y9", "4n6u4yb")
	require.NoError(t, err)
	require.Len(t, bodies, 3, "must be bulk-capable like task edit")
	for _, b := range bodies {
		assert.Equal(t, true, b["archived"])
	}

	bodies = nil
	_, _, err = runCLI(t, tf, "task", "unarchive", "4n6u4xw")
	require.NoError(t, err)
	require.Len(t, bodies, 1)
	assert.Equal(t, false, bodies[0]["archived"],
		"unarchive must send archived:false — this is why Archived has to be *bool, "+
			"not a bare bool with omitempty (which would serialise to nothing)")
}

// MANUAL: I archived 58 parents, then found three `to do` subtasks of an
// archived parent still live — ClickUp does not cascade, and I had not checked.
// They were only found because someone looked at the UI and pushed back.
func TestIncident_ArchiveDoesNotCascadeButReportsOrphans(t *testing.T) {
	t.Skip("Phase 3.9 — default no-cascade + loud orphan report")

	tf := testutil.NewTestFactory(t)
	var archived []string
	tf.HandleFunc("task/865d4q6bf", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			archived = append(archived, "865d4q6bf")
		}
		_, _ = io.WriteString(w, `{"id":"865d4q6bf","subtasks":[
			{"id":"865d4q0vh","status":{"status":"to do"}},
			{"id":"865d4q01q","status":{"status":"to do"}},
			{"id":"860qq4xkx","status":{"status":"to do"}}]}`)
	})

	// Default: only the named task is touched.
	_, errOut, err := runCLI(t, tf, "task", "archive", "865d4q6bf")
	require.NoError(t, err)
	assert.Equal(t, []string{"865d4q6bf"}, archived,
		"default must archive only what was named")
	assert.Contains(t, errOut, "3 subtasks", "must report the orphans it left behind")
	assert.Contains(t, errOut, "--cascade", "must name the flag that would include them")
	assert.Contains(t, errOut, "865d4q0vh", "must list orphan IDs, not just a count")
}

// ---------------------------------------------------------------------------
// Phase 5 — stop new API surface from going missing
// ---------------------------------------------------------------------------

// The exact regression that caused all of the above: a command reaching for a
// hand-written request type instead of the generated one, and thereby
// inheriting whatever fields that struct happens to lack.
func TestIncident_NoCommandUsesHandWrittenRequestTypes(t *testing.T) {
	t.Skip("Phase 5.11 — enable once Phase 4 lands")

	hits := grepRepo(t, "pkg/", `clickup\.\w+Request\b`)
	assert.Empty(t, hits,
		"commands must build requests from generated types; the hand-written "+
			"structs are how archived/locations went missing")
}

// A field present in the spec must reach the CLI without anyone writing Go.
// This is the acceptance criterion for the gen-api refactor.
func TestIncident_SpecFieldBecomesFlagWithoutHandCoding(t *testing.T) {
	t.Skip("Phase 5.12 — gen-api emits flag binding per request field")

	tf := testutil.NewTestFactory(t)
	out, _, err := runCLI(t, tf, "task", "edit", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "--archived",
		"every request field in the spec should surface as a flag automatically")
}

func TestIncident_NoHandRolledOperations(t *testing.T) {
	t.Skip("Phase 5.12 — coverage report from gen-api")

	hits := grepRepo(t, "pkg/", `http\.NewRequest|client\.Do\(`)
	assert.Empty(t, hits,
		"raw HTTP in a command means an operation was hand-rolled; use the generated one")
}
