package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestHandleErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantNil    bool
		wantMsg    string
		wantStatus int
	}{
		{
			name:    "200 returns nil",
			status:  200,
			body:    `{}`,
			wantNil: true,
		},
		{
			name:       "401 overrides message",
			status:     401,
			body:       `{"err": "something"}`,
			wantMsg:    "Authentication failed",
			wantStatus: 401,
		},
		{
			name:       "403 with message from API",
			status:     403,
			body:       `{"message": "You need admin access"}`,
			wantMsg:    "You need admin access",
			wantStatus: 403,
		},
		{
			name:       "403 without message uses default",
			status:     403,
			body:       `{}`,
			wantMsg:    "permission",
			wantStatus: 403,
		},
		{
			name:       "404 without message uses default",
			status:     404,
			body:       `{}`,
			wantMsg:    "not found",
			wantStatus: 404,
		},
		{
			name:       "429 rate limit",
			status:     429,
			body:       `{}`,
			wantMsg:    "Rate limit",
			wantStatus: 429,
		},
		{
			name:       "500 with err field",
			status:     500,
			body:       `{"err": "internal error"}`,
			wantStatus: 500,
		},
		{
			name:       "malformed JSON body",
			status:     500,
			body:       `{not json`,
			wantStatus: 500,
		},
		{
			name:       "empty body",
			status:     500,
			body:       ``,
			wantStatus: 500,
		},
		{
			name:       "body with err + message + ECODE",
			status:     400,
			body:       `{"err": "bad", "message": "Bad request", "ECODE": "ERR_001"}`,
			wantMsg:    "Bad request",
			wantStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := makeHTTPResponse(tt.status, tt.body)
			err := HandleErrorResponse(resp)

			if tt.wantNil {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.StatusCode)
			if tt.wantMsg != "" {
				assert.Contains(t, apiErr.Error(), tt.wantMsg)
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	t.Run("with Message", func(t *testing.T) {
		err := &APIError{StatusCode: 404, Message: "Task not found"}
		assert.Contains(t, err.Error(), "404")
		assert.Contains(t, err.Error(), "Task not found")
	})

	t.Run("with Err only", func(t *testing.T) {
		err := &APIError{StatusCode: 500, Err: "internal_error"}
		assert.Contains(t, err.Error(), "500")
		assert.Contains(t, err.Error(), "internal_error")
	})
}

func TestAuthExpiredError_Error(t *testing.T) {
	t.Run("with Detail", func(t *testing.T) {
		err := &AuthExpiredError{Detail: "Token expired at 2024-01-01"}
		assert.Contains(t, err.Error(), "Token expired")
	})

	t.Run("without Detail", func(t *testing.T) {
		err := &AuthExpiredError{}
		assert.Contains(t, err.Error(), "re-authenticate")
	})
}
