package clickup

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDate_RoundTrip_String(t *testing.T) {
	input := `"1700000000000"`
	var d Date
	if err := json.Unmarshal([]byte(input), &d); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if d.null {
		t.Fatal("expected non-null date")
	}
	want := time.UnixMilli(1700000000000)
	if !d.time.Equal(want) {
		t.Errorf("got time %v, want %v", d.time, want)
	}

	out, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(out) != "1700000000000" {
		t.Errorf("got %s, want 1700000000000", string(out))
	}
}

func TestDate_RoundTrip_Numeric(t *testing.T) {
	input := `1700000000000`
	var d Date
	if err := json.Unmarshal([]byte(input), &d); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	want := time.UnixMilli(1700000000000)
	if !d.time.Equal(want) {
		t.Errorf("got time %v, want %v", d.time, want)
	}
}

func TestDate_Null(t *testing.T) {
	nd := NullDate()
	out, err := json.Marshal(nd)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(out) != "null" {
		t.Errorf("got %s, want null", string(out))
	}
}

func TestDate_EmptyString(t *testing.T) {
	input := `""`
	var d Date
	if err := json.Unmarshal([]byte(input), &d); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if !d.null {
		t.Error("expected null date for empty string")
	}
}

func TestNewDate(t *testing.T) {
	now := time.Now()
	d := NewDate(now)
	if d.null {
		t.Fatal("expected non-null date")
	}
	if !d.time.Equal(now) {
		t.Errorf("got %v, want %v", d.time, now)
	}
}

func TestDate_Equal(t *testing.T) {
	a := NewDate(time.UnixMilli(1000))
	b := NewDate(time.UnixMilli(1000))
	c := NewDate(time.UnixMilli(2000))

	if !a.Equal(*b) {
		t.Error("expected a == b")
	}
	if a.Equal(*c) {
		t.Error("expected a != c")
	}

	null1 := NullDate()
	null2 := NullDate()
	if !null1.Equal(*null2) {
		t.Error("expected null == null")
	}
	if null1.Equal(*a) {
		t.Error("expected null != a")
	}
}
