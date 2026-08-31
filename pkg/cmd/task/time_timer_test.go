package task

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

func TestNewCmdTimeStart_Flags(t *testing.T) {
	cmd := NewCmdTimeStart(nil)
	assert.Equal(t, "start [<task-id>]", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("description"))
	assert.NotNil(t, cmd.Flags().Lookup("billable"))
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestTimeStart(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("POST", "team/12345/time_entries/start", 200, `{
		"data": {"id": "te1", "task": null, "wid": "12345", "start": "1700000000000", "duration": -1}
	}`)

	cmd := NewCmdTimeStart(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Timer started")
	assert.Contains(t, out, "abc123")
}

func TestTimeStart_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("POST", "team/12345/time_entries/start", 200, `{
		"data": {"id": "te1", "task": null, "wid": "12345", "start": "1700000000000", "duration": -1}
	}`)

	cmd := NewCmdTimeStart(tf.Factory)
	err := testutil.RunCommand(t, cmd, "abc123", "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.NotNil(t, parsed["data"])
}

func TestNewCmdTimeStop_Flags(t *testing.T) {
	cmd := NewCmdTimeStop(nil)
	assert.Equal(t, "stop", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestTimeStop(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("POST", "team/12345/time_entries/stop", 200, `{
		"data": {"id": "te1", "task": {"id": "abc123", "name": "My Task"}, "duration": 3600000}
	}`)

	cmd := NewCmdTimeStop(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Timer stopped")
	assert.Contains(t, out, "1h")
}

func TestNewCmdTimeRunning_Flags(t *testing.T) {
	cmd := NewCmdTimeRunning(nil)
	assert.Equal(t, "running", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestTimeRunning_NoTimer(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "team/12345/time_entries/current", 200, `{
		"data": {"id": "", "task": {"id": "", "name": ""}, "start": "", "duration": 0, "description": ""}
	}`)

	cmd := NewCmdTimeRunning(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "No timer is currently running")
}

func TestTimeRunning_Active(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "team/12345/time_entries/current", 200, `{
		"data": {"id": "te1", "task": {"id": "abc123", "name": "My Task"}, "start": "1700000000000", "duration": -1, "description": "Working"}
	}`)

	cmd := NewCmdTimeRunning(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Running timer")
	assert.Contains(t, out, "te1")
	assert.Contains(t, out, "My Task")
	assert.Contains(t, out, "Working")
}

func TestTimeSubcommands_Include_Timer(t *testing.T) {
	cmd := NewCmdTime(nil)
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	assert.True(t, names["start"], "expected 'start' subcommand")
	assert.True(t, names["stop"], "expected 'stop' subcommand")
	assert.True(t, names["running"], "expected 'running' subcommand")
}

// The stop response's duration comes back as a string on some workspaces.
func TestTimeStop_StringDuration(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("POST", "team/12345/time_entries/stop", 200, `{
		"data": {"id": "te1", "task": {"id": "abc123", "name": "My Task"}, "duration": "3600000"}
	}`)

	cmd := NewCmdTimeStop(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Timer stopped")
	assert.Contains(t, out, "1h")
}

// Likewise for the running-timer endpoint.
func TestTimeRunning_StringDuration(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "team/12345/time_entries/current", 200, `{
		"data": {"id": "te1", "task": {"id": "abc123", "name": "My Task"}, "start": "1700000000000", "duration": "-1", "description": "Working"}
	}`)

	cmd := NewCmdTimeRunning(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Running timer")
	assert.Contains(t, out, "My Task")
}

// A timer stopped inside a minute still reports a duration rather than a blank:
// "Timer stopped —  logged" reads like something went wrong. (It rounds to 0m —
// this pins the reporting, not the precision.)
func TestTimeStop_ShortDurationStillReportsATime(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("POST", "team/12345/time_entries/stop", 200, `{
		"data": {"id": "te1", "task": {"id": "abc123", "name": "My Task"}, "duration": 30000}
	}`)

	cmd := NewCmdTimeStop(tf.Factory)
	require.NoError(t, testutil.RunCommand(t, cmd))

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Timer stopped")
	assert.Contains(t, out, "0m")
}

// The stop response mixes types across its own timestamp fields — the spec's
// example shows a string start beside a numeric end. A failure here is
// especially bad: the timer is already stopped server-side, so the user sees an
// error and is then told no timer is running.
func TestTimeStop_MixedTimestampTypes(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("POST", "team/12345/time_entries/stop", 200, `{
		"data": {"id": "te1", "task": {"id": "abc123", "name": "My Task"},
		         "duration": 3600000, "start": "1595289395842", "end": 1595289452790, "at": "1595289452790"}
	}`)

	cmd := NewCmdTimeStop(tf.Factory)
	require.NoError(t, testutil.RunCommand(t, cmd))
	assert.Contains(t, tf.OutBuf.String(), "1h")
}

// The running-timer endpoint reads start back to compute elapsed time, so it
// breaks on a numeric start the same way.
func TestTimeRunning_NumericStart(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "team/12345/time_entries/current", 200, `{
		"data": {"id": "te1", "task": {"id": "abc123", "name": "My Task"}, "start": 1700000000000, "duration": -1, "description": "Working"}
	}`)

	cmd := NewCmdTimeRunning(tf.Factory)
	require.NoError(t, testutil.RunCommand(t, cmd))

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Running timer")
	assert.Contains(t, out, "elapsed")
}

// The timer commands emit their millisecond fields as JSON numbers. This is a
// deliberate change: `start` was a quoted string on these endpoints while `end`
// and `at` were numbers, and the response could not decode at all when ClickUp
// swapped them. One numeric shape across the command group is the contract now,
// so pin it rather than let it drift again.
func TestTimerCommands_JSONEmitsNumericMillis(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("POST", "team/12345/time_entries/stop", 200, `{
		"data": {"id": "te1", "task": {"id": "abc123", "name": "My Task"},
		         "duration": "3600000", "start": "1700000000000", "end": 1700003600000}
	}`)

	cmd := NewCmdTimeStop(tf.Factory)
	require.NoError(t, testutil.RunCommand(t, cmd, "--json"))

	var parsed struct {
		Data struct {
			Duration json.Number `json:"duration"`
			Start    json.Number `json:"start"`
			End      json.Number `json:"end"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(tf.OutBuf.Bytes(), &parsed))
	assert.Equal(t, "3600000", parsed.Data.Duration.String())
	assert.Equal(t, "1700000000000", parsed.Data.Start.String())
	assert.Equal(t, "1700003600000", parsed.Data.End.String())
}
