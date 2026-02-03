package cmdutil

import "errors"

// SilentError is an error that should not be printed (the command already handled output).
type SilentError struct {
	Err error
}

func (e *SilentError) Error() string {
	return e.Err.Error()
}

func (e *SilentError) Unwrap() error {
	return e.Err
}

// AuthError indicates an authentication failure.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "authentication required. Run 'clickup auth login' to authenticate"
}

// IsSilentError checks if the error is a SilentError.
func IsSilentError(err error) bool {
	var se *SilentError
	return errors.As(err, &se)
}

// IsAuthError checks if the error is an AuthError.
func IsAuthError(err error) bool {
	var ae *AuthError
	return errors.As(err, &ae)
}
