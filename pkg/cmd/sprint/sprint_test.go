package sprint

import (
	"testing"
	"time"
)

func TestParseMSTimestamp(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		isZero bool
		wantMS int64 // expected UnixMilli value, ignored when isZero is true
	}{
		{
			name:   "valid timestamp",
			input:  "1700000000000",
			isZero: false,
			wantMS: 1700000000000,
		},
		{
			name:   "empty string",
			input:  "",
			isZero: true,
		},
		{
			name:   "zero string",
			input:  "0",
			isZero: true,
		},
		{
			name:   "invalid non-numeric",
			input:  "not-a-number",
			isZero: true,
		},
		{
			name:   "another valid timestamp",
			input:  "1609459200000",
			isZero: false,
			wantMS: 1609459200000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMSTimestamp(tt.input)
			if tt.isZero {
				if !got.IsZero() {
					t.Errorf("parseMSTimestamp(%q) = %v, want zero time", tt.input, got)
				}
			} else {
				if got.IsZero() {
					t.Fatalf("parseMSTimestamp(%q) returned zero time, want non-zero", tt.input)
				}
				if got.UnixMilli() != tt.wantMS {
					t.Errorf("parseMSTimestamp(%q).UnixMilli() = %d, want %d", tt.input, got.UnixMilli(), tt.wantMS)
				}
			}
		})
	}
}

func TestFormatDateRange(t *testing.T) {
	jan1 := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	jan15 := time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC)
	zero := time.Time{}

	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  string
	}{
		{
			name:  "both dates set",
			start: jan1,
			end:   jan15,
			want:  "Jan 01 - Jan 15",
		},
		{
			name:  "only start set",
			start: jan1,
			end:   zero,
			want:  "started Jan 01",
		},
		{
			name:  "only end set",
			start: zero,
			end:   jan15,
			want:  "ends Jan 15",
		},
		{
			name:  "neither date set",
			start: zero,
			end:   zero,
			want:  "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDateRange(tt.start, tt.end)
			if got != tt.want {
				t.Errorf("formatDateRange(%v, %v) = %q, want %q", tt.start, tt.end, got, tt.want)
			}
		})
	}
}

func TestClassifySprint(t *testing.T) {
	// Fixed reference times for deterministic tests.
	jan1 := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	jan15 := time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC)
	jan31 := time.Date(2025, time.January, 31, 0, 0, 0, 0, time.UTC)
	feb15 := time.Date(2025, time.February, 15, 0, 0, 0, 0, time.UTC)
	zero := time.Time{}

	tests := []struct {
		name  string
		start time.Time
		due   time.Time
		now   time.Time
		want  string
	}{
		{
			name:  "complete - now is after due date",
			start: jan1,
			due:   jan15,
			now:   jan31,
			want:  "complete",
		},
		{
			name:  "in progress - now is between start and due",
			start: jan1,
			due:   jan31,
			now:   jan15,
			want:  "in progress",
		},
		{
			name:  "in progress - now equals start date",
			start: jan15,
			due:   jan31,
			now:   jan15,
			want:  "in progress",
		},
		{
			name:  "in progress - now equals due date",
			start: jan1,
			due:   jan15,
			now:   jan15,
			want:  "in progress",
		},
		{
			name:  "upcoming - now is before start",
			start: feb15,
			due:   time.Date(2025, time.February, 28, 0, 0, 0, 0, time.UTC),
			now:   jan31,
			want:  "upcoming",
		},
		{
			name:  "unknown - both dates zero",
			start: zero,
			due:   zero,
			now:   jan15,
			want:  "unknown",
		},
		{
			name:  "unknown - only due is zero and now is after start",
			start: jan1,
			due:   zero,
			now:   jan15,
			want:  "unknown",
		},
		{
			name:  "upcoming - only due is zero but now is before start",
			start: feb15,
			due:   zero,
			now:   jan15,
			want:  "upcoming",
		},
		{
			name:  "complete - only start is zero but due is set and past",
			start: zero,
			due:   jan15,
			now:   jan31,
			want:  "complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySprint(tt.start, tt.due, tt.now)
			if got != tt.want {
				t.Errorf("classifySprint(start=%v, due=%v, now=%v) = %q, want %q",
					tt.start, tt.due, tt.now, got, tt.want)
			}
		})
	}
}
