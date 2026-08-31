package task

import (
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
