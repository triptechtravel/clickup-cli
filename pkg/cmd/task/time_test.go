package task

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

func TestNewCmdTime_Subcommands(t *testing.T) {
	cmd := NewCmdTime(nil)
	assert.Equal(t, "time <command>", cmd.Use)
	assert.True(t, cmd.HasSubCommands())

	// Verify all expected subcommands are registered.
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	assert.True(t, names["log"], "expected 'log' subcommand")
	assert.True(t, names["list"], "expected 'list' subcommand")
	assert.True(t, names["delete"], "expected 'delete' subcommand")
}

func TestNewCmdTimeLog_Flags(t *testing.T) {
	cmd := NewCmdTimeLog(nil)
	assert.NotNil(t, cmd.Flags().Lookup("duration"))
	assert.NotNil(t, cmd.Flags().Lookup("description"))
	assert.NotNil(t, cmd.Flags().Lookup("date"))
	assert.NotNil(t, cmd.Flags().Lookup("billable"))
	assert.Equal(t, "log [<task-id>]", cmd.Use)
}

func TestNewCmdTimeList_Flags(t *testing.T) {
	cmd := NewCmdTimeList(nil)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("start-date"))
	assert.NotNil(t, cmd.Flags().Lookup("end-date"))
	assert.NotNil(t, cmd.Flags().Lookup("assignee"))
	assert.Equal(t, "list [<task-id>]", cmd.Use)
}

func TestNewCmdTimeDelete_Flags(t *testing.T) {
	cmd := NewCmdTimeDelete(nil)
	assert.NotNil(t, cmd.Flags().Lookup("yes"))
	assert.Equal(t, "delete <entry-id>", cmd.Use)

	// Verify shorthand -y.
	f := cmd.Flags().ShorthandLookup("y")
	assert.NotNil(t, f)
	assert.Equal(t, "yes", f.Name)
}

func TestFormatDuration(t *testing.T) {
	assert.Equal(t, "2h", formatDuration("7200000"))
	assert.Equal(t, "30m", formatDuration("1800000"))
	assert.Equal(t, "1h 30m", formatDuration("5400000"))
	assert.Equal(t, "0m", formatDuration("0"))
	assert.Equal(t, "invalid", formatDuration("invalid"))
}

// ClickUp returns a time entry's duration as a string ("3600000") on the time
// entries endpoints, but a number on others — and the running-timer sentinel is
// a negative number. Whichever arrives, the listing must render rather than
// fail.
func TestTimeList_MixedDurationTypes(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/time_entries", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		_, _ = w.Write([]byte(`{"data": [
			{"id": "te1", "duration": "3600000", "description": "String duration", "user": {"username": "alice"}},
			{"id": "te2", "duration": 1800000, "description": "Numeric duration", "user": {"username": "alice"}}
		]}`))
	})

	cmd := NewCmdTimeList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "1h")
	assert.Contains(t, out, "30m")
}

// The timesheet total is a billing number: an entry it cannot parse must not be
// dropped from the sum while still being counted in "across N entries". Floats
// are the form ClickUp is known to send once time is tracked.
func TestTimeList_TimesheetTotalCountsEveryEntry(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/time_entries", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		_, _ = w.Write([]byte(`{"data": [
			{"id": "te1", "duration": "3600000", "start": "1700000000000", "description": "String", "user": {"username": "alice"}},
			{"id": "te2", "duration": 1800000.0, "start": 1700000000000, "description": "Float", "user": {"username": "alice"}}
		]}`))
	})

	tf.Handle("GET", "user", 200, `{"user": {"id": 54874661, "username": "alice"}}`)

	cmd := NewCmdTimeList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--start-date", "2026-08-01", "--end-date", "2026-08-31")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "30m", "a float duration must render as a duration, not raw milliseconds")
	assert.NotContains(t, out, "1800000.0", "raw milliseconds must never reach the duration column")
	assert.Contains(t, out, "Total: 1h 30m", "every rendered entry must be in the total")
}

// `task time list --json` emits the same numeric shape as its sibling timer
// commands, whichever form the API sent. Before, the two halves of one command
// group disagreed: a duration was a quoted string here and a number from
// `task time stop`, so no single jq filter worked across them.
func TestTimeList_JSONEmitsNumericMillis(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/time_entries", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		_, _ = w.Write([]byte(`{"data": [
			{"id": "te1", "duration": "3600000", "start": "1700000000000", "end": 1700003600000}
		]}`))
	})

	cmd := NewCmdTimeList(tf.Factory)
	require.NoError(t, testutil.RunCommand(t, cmd, "abc123", "--json"))

	var entries []struct {
		Duration json.Number `json:"duration"`
		Start    json.Number `json:"start"`
		End      json.Number `json:"end"`
	}
	require.NoError(t, json.Unmarshal(tf.OutBuf.Bytes(), &entries))
	require.Len(t, entries, 1)
	assert.Equal(t, "3600000", entries[0].Duration.String())
	assert.Equal(t, "1700000000000", entries[0].Start.String())
	assert.Equal(t, "1700003600000", entries[0].End.String())
}
