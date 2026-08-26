package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// abortOnce serves an aborted connection for the first n requests, then 200.
// Aborting mid-flight is what a header timeout or a dropped connection looks
// like to the client.
func abortOnce(t *testing.T, n int32) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) <= n {
			panic(http.ErrAbortHandler)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &attempts
}

// A read that drops is safe to repeat, and repeating it is the difference
// between a working command and one the user has to run again by hand.
func TestRetry_GetRecoversFromATransientNetworkFailure(t *testing.T) {
	srv, attempts := abortOnce(t, 1)
	client := NewTestClient(srv.URL)

	req, err := http.NewRequest("GET", srv.URL, nil)
	require.NoError(t, err)
	resp, err := client.DoRequest(req)

	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, int32(2), attempts.Load(), "request was not retried")
}

// A POST creates something. A timeout waiting for headers means the server may
// already have processed it, so repeating it risks a second comment, a second
// task, a second time entry. The CLI must not decide that for the user.
func TestRetry_PostIsNotRepeatedAndSaysWhy(t *testing.T) {
	srv, attempts := abortOnce(t, 1)
	client := NewTestClient(srv.URL)

	req, err := http.NewRequest("POST", srv.URL, bytes.NewReader([]byte(`{"a":1}`)))
	require.NoError(t, err)
	_, err = client.DoRequest(req)

	require.Error(t, err)
	assert.Equal(t, int32(1), attempts.Load(), "a POST was silently repeated")
	assert.Contains(t, strings.ToLower(err.Error()), "may already have been applied",
		"the error does not warn that the write may have landed: %v", err)
}

// PUT is idempotent — the same edit applied twice is the same edit — so it is
// safe to repeat, which is what the task edit that failed here needed.
func TestRetry_PutIsRepeatable(t *testing.T) {
	srv, attempts := abortOnce(t, 1)
	client := NewTestClient(srv.URL)

	req, err := http.NewRequest("PUT", srv.URL, bytes.NewReader([]byte(`{"a":1}`)))
	require.NoError(t, err)
	resp, err := client.DoRequest(req)

	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, int32(2), attempts.Load(), "an idempotent write was not retried")
}

// Retries are bounded; a server that is simply down must not be hammered.
func TestRetry_IsBounded(t *testing.T) {
	srv, attempts := abortOnce(t, 100)
	client := NewTestClient(srv.URL)

	req, err := http.NewRequest("GET", srv.URL, nil)
	require.NoError(t, err)
	_, err = client.DoRequest(req)

	assert.Error(t, err)
	assert.LessOrEqual(t, attempts.Load(), int32(maxAttempts), "retried past the bound")
}

// A cancelled context stops the retry loop rather than working through it.
func TestRetry_StopsOnContextCancellation(t *testing.T) {
	srv, attempts := abortOnce(t, 100)
	client := NewTestClient(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	require.NoError(t, err)
	_, err = client.DoRequest(req)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || attempts.Load() <= 1,
		"kept retrying after cancellation (attempts=%d, err=%v)", attempts.Load(), err)
}

// A 5xx is the server saying "not now", and is worth one more go on a read.
func TestRetry_RetriesServerErrorsOnReads(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		if attempts.Add(1) == 1 {
			w.WriteHeader(502)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := NewTestClient(srv.URL)
	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := client.DoRequest(req)

	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, int32(2), attempts.Load())
}
