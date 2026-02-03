package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDate(t *testing.T) {
	d, err := parseDate("2025-06-15")
	assert.NoError(t, err)
	assert.NotNil(t, d)

	_, err = parseDate("invalid")
	assert.Error(t, err)

	_, err = parseDate("15-06-2025")
	assert.Error(t, err)
}

func TestParseDuration(t *testing.T) {
	ms, err := parseDuration("2h")
	assert.NoError(t, err)
	assert.Equal(t, 7200000, ms)

	ms, err = parseDuration("30m")
	assert.NoError(t, err)
	assert.Equal(t, 1800000, ms)

	ms, err = parseDuration("1h30m")
	assert.NoError(t, err)
	assert.Equal(t, 5400000, ms)

	_, err = parseDuration("invalid")
	assert.Error(t, err)
}

func TestFormatMillisDuration(t *testing.T) {
	assert.Equal(t, "2h", formatMillisDuration(7200000))
	assert.Equal(t, "30m", formatMillisDuration(1800000))
	assert.Equal(t, "1h 30m", formatMillisDuration(5400000))
	assert.Equal(t, "", formatMillisDuration(0))
	assert.Equal(t, "", formatMillisDuration(-1))
	assert.Equal(t, "< 1m", formatMillisDuration(500))
}
