package clickup

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Millis is a duration in milliseconds that decodes from either a JSON number
// or a JSON string.
//
// ClickUp is not consistent about which it sends: in a single task response
// time_spent comes back as the number 0 for untracked tasks and as the string
// "2040000" for tracked ones. Typing these fields as int64 meant one tracked
// subtask aborted the decode of its whole parent — name, status and every
// sibling included (issue #27).
//
// It marshals back as a number, which is what the field has always been in this
// CLI's own JSON output.
type Millis int64

// Int64 returns the duration in milliseconds.
func (m Millis) Int64() int64 { return int64(m) }

// UnmarshalJSON implements json.Unmarshaler.
func (m *Millis) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" || s == "" {
		*m = 0
		return nil
	}

	if strings.HasPrefix(s, `"`) {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		s = strings.TrimSpace(str)
		if s == "" {
			*m = 0
			return nil
		}
	}

	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		*m = Millis(n)
		return nil
	}
	// Some fields arrive as floats ("2040000.0"); truncate rather than fail.
	//
	// ParseFloat is looser than JSON: it also accepts Inf, NaN and magnitudes
	// int64 cannot hold, and converting one of those is implementation-defined
	// in Go — the same response would decode to MaxInt64 on arm64 and MinInt64
	// on amd64. A value that cannot be represented is an error, not a number.
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if !math.IsNaN(f) && !math.IsInf(f, 0) && f >= math.MinInt64 && f < math.MaxInt64 {
			*m = Millis(int64(f))
			return nil
		}
	}

	return fmt.Errorf("clickup: cannot decode %s as milliseconds", string(b))
}

// MarshalJSON implements json.Marshaler with a value receiver so encoding/json
// calls it on non-pointer struct fields.
func (m Millis) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(m), 10)), nil
}
