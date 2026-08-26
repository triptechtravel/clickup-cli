package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// maxAttempts bounds how many times one request is sent, including the first.
// A server that is simply down should not be hammered.
const maxAttempts = 3

// retryBackoff is the pause before each retry.
var retryBackoff = []time.Duration{250 * time.Millisecond, 1 * time.Second}

// UnsafeToRetryError reports a request that failed in a way that leaves its
// outcome unknown.
//
// A timeout waiting for response headers does not mean the server did nothing:
// it may have created the comment, the task, the time entry, and then failed to
// tell us. Repeating that automatically risks doing it twice, so the CLI says
// what it knows and lets the user check.
type UnsafeToRetryError struct {
	Method string
	Err    error
}

func (e *UnsafeToRetryError) Error() string {
	return fmt.Sprintf("%s request failed without a response (%v); it may already have been applied — check before retrying", e.Method, e.Err)
}

func (e *UnsafeToRetryError) Unwrap() error { return e.Err }

// repeatable reports whether sending a request again is safe.
//
// GET and HEAD change nothing. PUT and DELETE are idempotent by HTTP
// semantics — the same edit applied twice is the same edit — which covers
// `task edit`, one of the two calls that failed here. POST is not: it creates,
// and this is exactly where an automatic retry produces duplicates.
func repeatable(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
	default:
		return false
	}
	// A body that cannot be rewound cannot be sent twice.
	return req.Body == nil || req.GetBody != nil
}

// retryableStatus reports whether a response is worth another attempt.
// 429 is handled separately, by the rate limiter.
func retryableStatus(code int) bool {
	return code == http.StatusBadGateway ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout
}

// rewind returns a request ready to be sent again.
func rewind(req *http.Request) (*http.Request, error) {
	if req.GetBody == nil {
		return req, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	clone.Body = body
	return clone, nil
}

// waitBeforeRetry pauses, or gives up early if the caller has stopped caring.
func waitBeforeRetry(ctx context.Context, attempt int) error {
	d := retryBackoff[len(retryBackoff)-1]
	if attempt-1 < len(retryBackoff) {
		d = retryBackoff[attempt-1]
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// drain discards and closes a response body so the connection can be reused.
func drain(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
}

// send performs one request with bounded retries for transient failures.
func (t *authTransport) send(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 1; ; attempt++ {
		attemptReq := req
		if attempt > 1 {
			var err error
			if attemptReq, err = rewind(req); err != nil {
				return nil, err
			}
		}

		resp, err := t.base.RoundTrip(attemptReq)
		if err == nil && !retryableStatus(resp.StatusCode) {
			return resp, nil
		}

		if err != nil {
			lastErr = err
			// The caller stopped waiting; do not dress that up as a failure to
			// reach ClickUp.
			if ctxErr := req.Context().Err(); ctxErr != nil {
				return nil, ctxErr
			}
			if !repeatable(req) {
				return nil, &UnsafeToRetryError{Method: req.Method, Err: err}
			}
		} else {
			lastErr = fmt.Errorf("HTTP %d from ClickUp", resp.StatusCode)
			if !repeatable(req) {
				return resp, nil
			}
			drain(resp)
		}

		if attempt >= maxAttempts {
			if err == nil {
				// Out of attempts on a retryable status: hand the response back
				// so the caller reports ClickUp's own error rather than ours.
				return t.base.RoundTrip(req)
			}
			return nil, fmt.Errorf("after %d attempts: %w", attempt, lastErr)
		}
		if waitErr := waitBeforeRetry(req.Context(), attempt); waitErr != nil {
			return nil, errors.Join(lastErr, waitErr)
		}
	}
}
