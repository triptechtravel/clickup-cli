package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
