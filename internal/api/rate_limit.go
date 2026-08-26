package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// maxWait caps how long a single backoff may last.
//
// The reset header is documented in seconds, but a value in millis computes a
// wait measured in decades, and nothing else bounds it. A ceiling turns a bad
// header into a slow request rather than a hung CLI.
const maxWait = 90 * time.Second

// RateLimiter tracks ClickUp API rate limits and provides backoff.
type RateLimiter struct {
	mu        sync.Mutex
	remaining int
	resetAt   time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{remaining: 100}
}

// Update reads rate limit headers from a response.
func (rl *RateLimiter) Update(resp *http.Response) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if v := resp.Header.Get("X-RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.remaining = n
		}
	}
	if v := resp.Header.Get("X-RateLimit-Reset"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			rl.resetAt = time.Unix(ts, 0)
		}
	}
}

// Wait blocks until it's safe to make another request.
//
// Deprecated: prefer WaitContext. This cannot be interrupted, so every deadline
// above it is advisory.
func (rl *RateLimiter) Wait() {
	_ = rl.WaitContext(context.Background())
}

// WaitContext blocks until it's safe to make another request, or until ctx ends.
//
// The uninterruptible version made a mockery of every timeout above it: with
// the quota exhausted, a search with a 90-second deadline ran for 126 seconds,
// because time.Sleep does not care about contexts and the sleep sits inside the
// RoundTripper, where http.Client.Timeout cannot reach it either.
func (rl *RateLimiter) WaitContext(ctx context.Context) error {
	rl.mu.Lock()
	remaining := rl.remaining
	resetAt := rl.resetAt
	rl.mu.Unlock()

	if remaining > 0 {
		return nil
	}

	waitDuration := time.Until(resetAt)
	if waitDuration <= 0 {
		return nil
	}
	if waitDuration > maxWait {
		waitDuration = maxWait
	}

	timer := time.NewTimer(waitDuration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("rate limit backoff interrupted: %w", ctx.Err())
	}
}

// ShouldRetry returns true if the response indicates rate limiting and the request should be retried.
func (rl *RateLimiter) ShouldRetry(resp *http.Response) bool {
	return resp.StatusCode == 429
}
