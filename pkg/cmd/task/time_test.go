package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdTime_Subcommands(t *testing.T) {
	cmd := NewCmdTime(nil)
	assert.Equal(t, "time <command>", cmd.Use)
	assert.True(t, cmd.HasSubCommands())
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

func TestFormatDuration(t *testing.T) {
	assert.Equal(t, "2h", formatDuration("7200000"))
	assert.Equal(t, "30m", formatDuration("1800000"))
	assert.Equal(t, "1h 30m", formatDuration("5400000"))
	assert.Equal(t, "0m", formatDuration("0"))
	assert.Equal(t, "invalid", formatDuration("invalid"))
}
