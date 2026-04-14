package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthTransport_InjectsHeaders verifies our authTransport adds the
// correct headers per the V2 spec's security requirements.
func TestAuthTransport_InjectsHeaders(t *testing.T) {
	var captured *http.Request

	server := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	})
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.DoRequest(req)

	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "test-token", captured.Header.Get("Authorization"))
	assert.True(t, strings.HasPrefix(captured.Header.Get("User-Agent"), "clickup-cli/"))
}

// TestAuthTransport_RetryOn429 verifies the single retry on rate limit.
func TestAuthTransport_RetryOn429(t *testing.T) {
	attempts := 0

	server := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "0")
			w.WriteHeader(429)
			w.Write([]byte(`{"err": "Rate limit exceeded"}`))
			return
		}
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok": true}`))
	})
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.DoRequest(req)

	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 2, attempts, "should retry once on 429")
}

// TestAuthTransport_429ThenAnother429 verifies no infinite retry.
func TestAuthTransport_429ThenAnother429(t *testing.T) {
	attempts := 0

	server := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "0")
		w.WriteHeader(429)
		w.Write([]byte(`{"err": "Rate limit exceeded"}`))
	})
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.DoRequest(req)

	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 429, resp.StatusCode, "should return 429 after retry fails")
	assert.Equal(t, 2, attempts, "should only retry once")
}

// TestAuthTransport_401WithoutECODE returns AuthExpiredError.
func TestAuthTransport_401WithoutECODE(t *testing.T) {
	server := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(401)
		w.Write([]byte(`{}`))
	})
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	_, err := client.DoRequest(req)

	require.Error(t, err)
	var authErr *AuthExpiredError
	assert.ErrorAs(t, err, &authErr, "401 without ECODE should be AuthExpiredError")
}

// TestAuthTransport_401WithECODE returns APIError (permission, not auth expiry).
func TestAuthTransport_401WithECODE(t *testing.T) {
	server := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(401)
		w.Write([]byte(`{"err": "Token invalid", "ECODE": "OAUTH_025"}`))
	})
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	_, err := client.DoRequest(req)

	require.Error(t, err)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "401 with ECODE should be APIError, not AuthExpiredError")
	assert.Equal(t, 401, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "ECODE")
}

// TestAuthTransport_401EmptyBody returns AuthExpiredError.
func TestAuthTransport_401EmptyBody(t *testing.T) {
	server := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(401)
	})
	defer server.Close()

	client := NewTestClient(server.URL)
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	_, err := client.DoRequest(req)

	require.Error(t, err)
	var authErr *AuthExpiredError
	assert.ErrorAs(t, err, &authErr, "401 with empty body should be AuthExpiredError")
}

// TestClient_URL verifies the URL helper builds correct paths.
func TestClient_URL(t *testing.T) {
	server := newTestServer(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	client := NewTestClient(server.URL)

	assert.Equal(t, server.URL+"/api/v2/task/abc123", client.URL("task/%s", "abc123"))
	assert.Equal(t, server.URL+"/api/v2/task/abc123/tag/bug", client.URL("task/%s/tag/%s", "abc123", "bug"))
	assert.Equal(t, server.URL+"/api/v2/space/123", client.URL("space/%s", "123"))
}

// TestClient_BaseURLV3 verifies that BaseURLV3 returns the v3 base derived from the v2 base.
func TestClient_BaseURLV3(t *testing.T) {
	server := newTestServer(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	client := NewTestClient(server.URL)

	v2 := client.BaseURL()
	v3 := client.BaseURLV3()

	assert.Equal(t, server.URL+"/api/v2", v2)
	assert.Equal(t, server.URL+"/api/v3", v3)
}

func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}
