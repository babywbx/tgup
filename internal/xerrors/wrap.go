package xerrors

import (
	"errors"
	"fmt"
)

// Error is an app-owned typed error wrapper.
type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Message == "" && e.Err == nil {
		return string(e.Code)
	}
	if e.Message == "" {
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
}

// Unwrap returns wrapped error.
func (e *Error) Unwrap() error { return e.Err }

// Wrap creates a typed error.
func Wrap(code Code, message string, err error) error {
	if err == nil && message == "" {
		return &Error{Code: code}
	}
	return &Error{Code: code, Message: message, Err: err}
}

// IsCode checks whether err contains Error with the given code.
func IsCode(err error, code Code) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Code == code
}
