package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ValidateToken checks that a token is valid by calling the ClickUp API.
// Returns the authenticated user info on success.
func ValidateToken(token string) (*AuthUser, error) {
	req, err := http.NewRequest("GET", "https://api.clickup.com/api/v2/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("invalid or expired token")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	var result struct {
		User AuthUser `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &result.User, nil
}

// AuthUser represents the authenticated ClickUp user.
type AuthUser struct {
	ID             int    `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	ProfilePicture string `json:"profilePicture"`
}
