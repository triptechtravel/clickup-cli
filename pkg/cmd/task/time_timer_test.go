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
