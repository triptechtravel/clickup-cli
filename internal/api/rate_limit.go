package api

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

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
func (rl *RateLimiter) Wait() {
	rl.mu.Lock()
	remaining := rl.remaining
	resetAt := rl.resetAt
	rl.mu.Unlock()

	if remaining > 0 {
		return
	}

	waitDuration := time.Until(resetAt)
	if waitDuration > 0 {
		time.Sleep(waitDuration)
	}
}

// ShouldRetry returns true if the response indicates rate limiting and the request should be retried.
func (rl *RateLimiter) ShouldRetry(resp *http.Response) bool {
	return resp.StatusCode == 429
}
