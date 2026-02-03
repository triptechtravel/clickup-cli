package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/raksul/go-clickup/clickup"
	"github.com/triptechtravel/clickup-cli/internal/build"
)

// Client wraps the go-clickup client with auth and rate limiting.
type Client struct {
	Clickup     *clickup.Client
	HTTPClient  *http.Client
	RateLimiter *RateLimiter
	token       string
}

// authTransport injects the Authorization header into every request.
type authTransport struct {
	token string
	base  http.RoundTripper
	rl    *RateLimiter
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.token)
	req.Header.Set("User-Agent", fmt.Sprintf("clickup-cli/%s", build.Version))

	t.rl.Wait()

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	t.rl.Update(resp)

	// Retry once on 429
	if t.rl.ShouldRetry(resp) {
		resp.Body.Close()
		t.rl.Wait()
		resp, err = t.base.RoundTrip(req)
		if err != nil {
			return resp, err
		}
		t.rl.Update(resp)
	}

	if resp.StatusCode == 401 {
		resp.Body.Close()
		return nil, &AuthExpiredError{}
	}

	return resp, nil
}

// NewClient creates a new API client with the given token.
func NewClient(token string) *Client {
	rl := NewRateLimiter()

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &authTransport{
			token: token,
			base:  http.DefaultTransport,
			rl:    rl,
		},
	}

	clickupClient := clickup.NewClient(httpClient, token)

	return &Client{
		Clickup:     clickupClient,
		HTTPClient:  httpClient,
		RateLimiter: rl,
		token:       token,
	}
}

// Token returns the API token used by this client.
func (c *Client) Token() string {
	return c.token
}

// DoRequest makes a raw HTTP request to the ClickUp API.
func (c *Client) DoRequest(req *http.Request) (*http.Response, error) {
	return c.HTTPClient.Do(req)
}
