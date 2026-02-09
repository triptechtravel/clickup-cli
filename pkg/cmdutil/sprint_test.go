package cmdutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseMSTimestamp(t *testing.T) {
	t.Run("valid timestamp", func(t *testing.T) {
		result := ParseMSTimestamp("1736899200000")
		assert.False(t, result.IsZero())
		assert.Equal(t, time.UnixMilli(1736899200000), result)
	})

	t.Run("empty string", func(t *testing.T) {
		result := ParseMSTimestamp("")
		assert.True(t, result.IsZero())
	})

	t.Run("zero string", func(t *testing.T) {
		result := ParseMSTimestamp("0")
		assert.True(t, result.IsZero())
	})

	t.Run("invalid string", func(t *testing.T) {
		result := ParseMSTimestamp("not-a-number")
		assert.True(t, result.IsZero())
	})
}
