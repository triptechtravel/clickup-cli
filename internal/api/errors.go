package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents an error returned by the ClickUp API.
type APIError struct {
	StatusCode int
	Message    string
	Err        string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("ClickUp API error (HTTP %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("ClickUp API error (HTTP %d): %s", e.StatusCode, e.Err)
}

// HandleErrorResponse checks an HTTP response for errors and returns a user-friendly error.
func HandleErrorResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	apiErr := &APIError{StatusCode: resp.StatusCode}

	var errorBody struct {
		Err     string `json:"err"`
		Message string `json:"message"`
		ECODE   string `json:"ECODE"`
	}
	if json.Unmarshal(body, &errorBody) == nil {
		apiErr.Err = errorBody.Err
		apiErr.Message = errorBody.Message
	}

	switch resp.StatusCode {
	case 401:
		apiErr.Message = "Authentication failed. Run 'clickup auth login' to re-authenticate."
	case 403:
		if apiErr.Message == "" {
			apiErr.Message = "You don't have permission to perform this action."
		}
	case 404:
		if apiErr.Message == "" {
			apiErr.Message = "Resource not found. Check the ID and try again."
		}
	case 429:
		if apiErr.Message == "" {
			apiErr.Message = "Rate limit exceeded. Please wait and try again."
		}
	}

	return apiErr
}
