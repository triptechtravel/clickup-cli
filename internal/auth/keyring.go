package auth

import (
	"fmt"

	"github.com/triptechtravel/clickup-cli/internal/config"
	"github.com/zalando/go-keyring"
)

const (
	serviceName = "clickup-cli"
	tokenKey    = "api_token"
	methodKey   = "auth_method"
)

// StoreToken saves the API token to the OS keyring, falling back to plaintext.
func StoreToken(token, method string) error {
	err := keyring.Set(serviceName, tokenKey, token)
	if err == nil {
		_ = keyring.Set(serviceName, methodKey, method)
		return nil
	}

	// Fallback to file-based storage
	ac := &config.AuthConfig{
		Token:      token,
		AuthMethod: method,
	}
	if saveErr := ac.Save(); saveErr != nil {
		return fmt.Errorf("failed to store token: keyring error: %w, file error: %v", err, saveErr)
	}
	fmt.Println("Warning: Could not use OS keyring. Token stored in plain text at", config.AuthFile())
	return nil
}

// GetToken retrieves the stored API token.
func GetToken() (string, error) {
	token, err := keyring.Get(serviceName, tokenKey)
	if err == nil && token != "" {
		return token, nil
	}

	// Fallback to file-based storage
	ac, err := config.LoadAuth()
	if err != nil {
		return "", fmt.Errorf("no stored credentials found: %w", err)
	}
	if ac.Token == "" {
		return "", fmt.Errorf("not authenticated. Run 'clickup auth login' to authenticate")
	}
	return ac.Token, nil
}

// GetAuthMethod returns the authentication method used ("oauth" or "token").
func GetAuthMethod() string {
	method, err := keyring.Get(serviceName, methodKey)
	if err == nil && method != "" {
		return method
	}

	ac, err := config.LoadAuth()
	if err != nil || ac.AuthMethod == "" {
		return "unknown"
	}
	return ac.AuthMethod
}

// ClearToken removes stored credentials from keyring and file.
func ClearToken() error {
	_ = keyring.Delete(serviceName, tokenKey)
	_ = keyring.Delete(serviceName, methodKey)

	ac := &config.AuthConfig{}
	_ = ac.Clear()
	return nil
}
