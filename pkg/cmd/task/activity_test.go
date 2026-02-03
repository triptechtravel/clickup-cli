package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdActivity_Usage(t *testing.T) {
	cmd := NewCmdActivity(nil)
	assert.Equal(t, "activity [<task-id>]", cmd.Use)
}

func TestParseUnixMillis(t *testing.T) {
	ts, err := parseUnixMillis("1706918400000")
	assert.NoError(t, err)
	assert.Equal(t, 2024, ts.Year())

	_, err = parseUnixMillis("invalid")
	assert.Error(t, err)
}
