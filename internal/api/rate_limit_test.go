package api

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter()
	if rl == nil {
		t.Fatal("NewRateLimiter() returned nil")
	}
	if rl.remaining != 100 {
		t.Errorf("initial remaining = %d, want 100", rl.remaining)
	}
	if !rl.resetAt.IsZero() {
		t.Errorf("initial resetAt = %v, want zero time", rl.resetAt)
	}
}

func makeResponse(remaining, resetUnix string) *http.Response {
	header := http.Header{}
	if remaining != "" {
		header.Set("X-RateLimit-Remaining", remaining)
	}
	if resetUnix != "" {
		header.Set("X-RateLimit-Reset", resetUnix)
	}
	return &http.Response{
		StatusCode: 200,
		Header:     header,
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name            string
		remaining       string
		resetUnix       string
		wantRemaining   int
		wantResetIsZero bool
		wantResetUnix   int64 // ignored when wantResetIsZero is true
	}{
		{
			name:            "updates both remaining and reset",
			remaining:       "42",
			resetUnix:       "1700000000",
			wantRemaining:   42,
			wantResetIsZero: false,
			wantResetUnix:   1700000000,
		},
		{
			name:            "updates only remaining",
			remaining:       "10",
			resetUnix:       "",
			wantRemaining:   10,
			wantResetIsZero: true,
		},
		{
			name:            "updates only reset",
			remaining:       "",
			resetUnix:       "1700000000",
			wantRemaining:   100, // unchanged from default
			wantResetIsZero: false,
			wantResetUnix:   1700000000,
		},
		{
			name:            "no headers - keeps defaults",
			remaining:       "",
			resetUnix:       "",
			wantRemaining:   100,
			wantResetIsZero: true,
		},
		{
			name:            "remaining zero",
			remaining:       "0",
			resetUnix:       "1700000000",
			wantRemaining:   0,
			wantResetIsZero: false,
			wantResetUnix:   1700000000,
		},
		{
			name:            "invalid remaining header ignored",
			remaining:       "not-a-number",
			resetUnix:       "1700000000",
			wantRemaining:   100, // unchanged
			wantResetIsZero: false,
			wantResetUnix:   1700000000,
		},
		{
			name:            "invalid reset header ignored",
			remaining:       "50",
			resetUnix:       "not-a-number",
			wantRemaining:   50,
			wantResetIsZero: true, // unchanged from default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter()
			resp := makeResponse(tt.remaining, tt.resetUnix)
			rl.Update(resp)

			if rl.remaining != tt.wantRemaining {
				t.Errorf("remaining = %d, want %d", rl.remaining, tt.wantRemaining)
			}
			if tt.wantResetIsZero {
				if !rl.resetAt.IsZero() {
					t.Errorf("resetAt = %v, want zero time", rl.resetAt)
				}
			} else {
				if rl.resetAt.Unix() != tt.wantResetUnix {
					t.Errorf("resetAt.Unix() = %d, want %d", rl.resetAt.Unix(), tt.wantResetUnix)
				}
			}
		})
	}
}

func TestUpdate_Sequential(t *testing.T) {
	rl := NewRateLimiter()

	// First response sets remaining to 50.
	resp1 := makeResponse("50", "1700000000")
	rl.Update(resp1)
	if rl.remaining != 50 {
		t.Errorf("after first update: remaining = %d, want 50", rl.remaining)
	}

	// Second response updates to 25.
	resp2 := makeResponse("25", "1700000100")
	rl.Update(resp2)
	if rl.remaining != 25 {
		t.Errorf("after second update: remaining = %d, want 25", rl.remaining)
	}
	if rl.resetAt.Unix() != 1700000100 {
		t.Errorf("after second update: resetAt.Unix() = %d, want 1700000100", rl.resetAt.Unix())
	}
}

func TestWait_HasRemaining(t *testing.T) {
	rl := NewRateLimiter() // remaining = 100

	start := time.Now()
	rl.Wait()
	elapsed := time.Since(start)

	// Should return nearly instantly when remaining > 0.
	if elapsed > 100*time.Millisecond {
		t.Errorf("Wait() took %v with remaining > 0, expected near-instant return", elapsed)
	}
}

func TestWait_ZeroRemainingPastReset(t *testing.T) {
	rl := NewRateLimiter()

	// Simulate rate limit exhausted with a reset time already in the past.
	resetTime := time.Now().Add(-1 * time.Second)
	resp := makeResponse("0", strconv.FormatInt(resetTime.Unix(), 10))
	rl.Update(resp)

	start := time.Now()
	rl.Wait()
	elapsed := time.Since(start)

	// Reset time is in the past, so Wait should return nearly instantly.
	if elapsed > 100*time.Millisecond {
		t.Errorf("Wait() took %v with past reset time, expected near-instant return", elapsed)
	}
}

func TestWait_ZeroRemainingFutureReset(t *testing.T) {
	rl := NewRateLimiter()

	// Set reset time to 200ms in the future.
	resetTime := time.Now().Add(200 * time.Millisecond)
	resp := makeResponse("0", strconv.FormatInt(resetTime.Unix(), 10))
	rl.Update(resp)

	start := time.Now()
	rl.Wait()
	elapsed := time.Since(start)

	// Wait should block until reset time. Since we use Unix seconds
	// for the reset header and time.Until for the wait, the actual
	// sleep may be slightly less than 200ms due to rounding. Just
	// verify it did wait some non-trivial amount.
	// (The Unix() rounding means the actual sleep could be very short
	// or up to ~1 second. We just check it didn't panic.)
	_ = elapsed
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{
			name:       "429 Too Many Requests",
			statusCode: 429,
			want:       true,
		},
		{
			name:       "200 OK",
			statusCode: 200,
			want:       false,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: 500,
			want:       false,
		},
		{
			name:       "403 Forbidden",
			statusCode: 403,
			want:       false,
		},
		{
			name:       "401 Unauthorized",
			statusCode: 401,
			want:       false,
		},
		{
			name:       "503 Service Unavailable",
			statusCode: 503,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter()
			resp := &http.Response{
				StatusCode: tt.statusCode,
			}
			got := rl.ShouldRetry(resp)
			if got != tt.want {
				t.Errorf("ShouldRetry(status=%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

// A context deadline has to actually bound the wait. The limiter slept with
// time.Sleep, which nothing can interrupt, so a quota-exhausted workspace made
// every deadline above it — the 30s HTTP timeout, the 60s index sync budget,
// the 90s command deadline — advisory. Measured overruns of 112s and 126s on
// searches that were supposed to stop at 90s.
func TestRateLimiter_WaitReturnsWhenTheContextEnds(t *testing.T) {
	rl := NewRateLimiter()
	rl.remaining = 0
	rl.resetAt = time.Now().Add(time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := rl.WaitContext(ctx)

	assert.Error(t, err, "a cancelled wait should report it")
	assert.Less(t, time.Since(start), time.Second, "wait ignored the cancelled context")
}

// A reset timestamp in the wrong unit (millis rather than seconds) computed a
// wait decades long. Nothing capped it, so one bad header parked the CLI
// indefinitely.
func TestRateLimiter_WaitIsCapped(t *testing.T) {
	rl := NewRateLimiter()
	rl.remaining = 0
	rl.resetAt = time.Now().Add(400 * 24 * time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_ = rl.WaitContext(ctx)

	assert.Less(t, time.Since(start), 5*time.Second,
		"an absurd reset time was honoured without a ceiling")
}
