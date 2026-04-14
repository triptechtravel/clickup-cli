package clickup

import (
	"encoding/json"
	"strconv"
	"time"
)

// Date is a timestamp type that handles both string and numeric JSON representations
// of Unix millisecond timestamps, as used by the ClickUp API.
type Date struct {
	unix json.Number
	time time.Time
	null bool
}

// NewDate creates a Date from a time.Time.
func NewDate(t time.Time) *Date {
	return &Date{
		unix: int64ToJsonNumber(t.UnixMilli()),
		time: t,
	}
}

// NewDateWithUnixTime creates a Date from a Unix millisecond timestamp.
func NewDateWithUnixTime(unix int64) *Date {
	return &Date{
		unix: int64ToJsonNumber(unix),
		time: time.UnixMilli(unix),
	}
}

// Time returns a pointer to the underlying time, or nil if null.
func (d Date) Time() *time.Time {
	if d.null {
		return nil
	}
	return &d.time
}

// String returns the string representation of the date.
func (d Date) String() string {
	if d.null {
		return ""
	}
	return d.time.String()
}

// Equal reports whether x and y are equal.
func (x Date) Equal(y Date) bool {
	if x.null {
		return x.null == y.null
	}
	return x.time.Equal(y.time)
}

// UnmarshalJSON implements json.Unmarshaler. It handles both string timestamps
// ("1700000000000") and numeric timestamps (1700000000000).
func (d *Date) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		if str == "" {
			d.null = true
			return nil
		}
	}

	var v json.Number
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	n, err := jsonNumberToInt64(v)
	if err != nil {
		return err
	}

	d.unix = v
	d.time = time.UnixMilli(n)
	d.null = false

	return nil
}

// MarshalJSON implements json.Marshaler.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.null {
		return json.Marshal(nil)
	}
	return json.Marshal(d.unix)
}

func int64ToJsonNumber(n int64) json.Number {
	b := []byte(strconv.Itoa(int(n)))

	var v json.Number
	if err := json.Unmarshal(b, &v); err != nil {
		panic(err.Error())
	}

	return v
}

func jsonNumberToInt64(num json.Number) (int64, error) {
	n, err := num.Int64()
	if err != nil {
		return 0, err
	}
	return n, nil
}
