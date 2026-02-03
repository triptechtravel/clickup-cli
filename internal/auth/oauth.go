package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// Default ClickUp OAuth app credentials. These can be overridden via config.
	DefaultClientID     = ""
	DefaultClientSecret = ""

	authorizeURL = "https://app.clickup.com/api"
	tokenURL     = "https://api.clickup.com/api/v2/oauth/token"
	redirectPath = "/callback"
)

// OAuthResult holds the result of an OAuth flow.
type OAuthResult struct {
	AccessToken string
	Error       error
}

// OAuthFlow performs the OAuth 2.0 authorization code flow using a local HTTP server.
func OAuthFlow(clientID, clientSecret string) (string, error) {
	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("OAuth client_id and client_secret are required.\n" +
			"Register an OAuth app at https://clickup.com/integrations and set them in config,\n" +
			"or use 'clickup auth login --token' to authenticate with a personal API token instead")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d%s", port, redirectPath)

	resultCh := make(chan OAuthResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			resultCh <- OAuthResult{Error: fmt.Errorf("authorization failed: %s", errMsg)}
			fmt.Fprintf(w, "<html><body><h2>Authorization failed</h2><p>%s</p><p>You can close this tab.</p></body></html>", errMsg)
			return
		}

		token, err := exchangeCode(code, clientID, clientSecret, redirectURI)
		if err != nil {
			resultCh <- OAuthResult{Error: err}
			fmt.Fprintf(w, "<html><body><h2>Token exchange failed</h2><p>%s</p><p>You can close this tab.</p></body></html>", err.Error())
			return
		}

		resultCh <- OAuthResult{AccessToken: token}
		fmt.Fprint(w, "<html><body><h2>✓ Authenticated with ClickUp!</h2><p>You can close this tab and return to the terminal.</p></body></html>")
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s", authorizeURL, url.QueryEscape(clientID), url.QueryEscape(redirectURI))

	fmt.Printf("Opening browser for ClickUp authorization...\n")
	fmt.Printf("If the browser doesn't open, visit this URL:\n  %s\n\n", authURL)
	fmt.Printf("Waiting for authorization...\n")

	// Return the auth URL so the caller can open it
	select {
	case result := <-resultCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		if result.Error != nil {
			return "", result.Error
		}
		return result.AccessToken, nil
	case <-time.After(5 * time.Minute):
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		return "", fmt.Errorf("authorization timed out after 5 minutes")
	}
}

// GetAuthURL returns the authorization URL for the OAuth flow.
func GetAuthURL(clientID, redirectURI string) string {
	return fmt.Sprintf("%s?client_id=%s&redirect_uri=%s", authorizeURL, url.QueryEscape(clientID), url.QueryEscape(redirectURI))
}

func exchangeCode(code, clientID, clientSecret, redirectURI string) (string, error) {
	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("token exchange error: %s", result.Error)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return result.AccessToken, nil
}
