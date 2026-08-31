package clickup

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ClickUp is inconsistent about millisecond fields: the same response can carry
// time_spent as a JSON number on one task and a JSON string on the next. Millis
// accepts either so one string cannot fail an entire decode.
func TestMillis_Unmarshal(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int64
	}{
		{"number", `0`, 0},
		{"number nonzero", `2040000`, 2040000},
		{"string", `"2040000"`, 2040000},
		{"empty string", `""`, 0},
		{"null", `null`, 0},
		{"negative number (running timer)", `-1`, -1},
		{"negative string", `"-1"`, -1},
		{"float milliseconds", `2040000.0`, 2040000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Millis
			require.NoError(t, json.Unmarshal([]byte(tt.in), &m))
			assert.Equal(t, tt.want, m.Int64())
		})
	}
}

func TestMillis_UnmarshalRejectsGarbage(t *testing.T) {
	var m Millis
	assert.Error(t, json.Unmarshal([]byte(`"two thousand"`), &m))
	assert.Error(t, json.Unmarshal([]byte(`{"ms":1}`), &m))
}

// The --json contract predates this type: time_spent has always been emitted as
// a JSON number, and normalising strings to numbers keeps `jq` filters working.
func TestMillis_MarshalsAsNumber(t *testing.T) {
	var m Millis
	require.NoError(t, json.Unmarshal([]byte(`"2040000"`), &m))

	b, err := json.Marshal(m)
	require.NoError(t, err)
	assert.JSONEq(t, `2040000`, string(b))

	b, err = json.Marshal(Millis(0))
	require.NoError(t, err)
	assert.Equal(t, `0`, string(b))
}

// Issue #27: a string time_spent on a nested subtask aborted the whole decode.
func TestTask_StringTimeSpentOnSubtask(t *testing.T) {
	const body = `{
		"id": "86cay82n2",
		"name": "Parent",
		"time_spent": 0,
		"time_estimate": "7200000",
		"subtasks": [
			{"id": "86caxfujb", "name": "A", "time_spent": 0},
			{"id": "86caygxwe", "name": "B", "time_spent": "2040000"}
		]
	}`

	var task Task
	require.NoError(t, json.Unmarshal([]byte(body), &task))

	assert.Equal(t, int64(7200000), task.TimeEstimate.Int64())
	require.Len(t, task.Subtasks, 2)
	assert.Equal(t, int64(0), task.Subtasks[0].TimeSpent.Int64())
	assert.Equal(t, int64(2040000), task.Subtasks[1].TimeSpent.Int64())
	assert.Equal(t, "B", task.Subtasks[1].Name)
}

// The float fallback exists for "2040000.0". It must not become a back door for
// values int64 cannot hold: converting an out-of-range float is
// implementation-defined in Go, so the same response would decode to MaxInt64
// on arm64 and MinInt64 on amd64.
func TestMillis_RejectsOutOfRangeAndNonFinite(t *testing.T) {
	for _, in := range []string{
		`1e30`,
		`99999999999999999999`,
		`-99999999999999999999`,
		`"Inf"`,
		`"-Inf"`,
		`"NaN"`,
		`"infinity"`,
	} {
		var m Millis
		assert.Error(t, json.Unmarshal([]byte(in), &m), "%s must not decode to a fabricated millisecond count", in)
	}
}

// math.MaxInt64 is not representable as a float64: it rounds to 2^63, which
// int64 cannot hold. The boundary has to exclude it.
func TestMillis_RejectsExactlyTwoToThe63(t *testing.T) {
	var m Millis
	assert.Error(t, json.Unmarshal([]byte(`9223372036854775808`), &m))

	require.NoError(t, json.Unmarshal([]byte(`9223372036854775807`), &m))
	assert.Equal(t, int64(9223372036854775807), m.Int64())
}
