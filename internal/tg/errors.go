package tg

import "errors"

var (
	// ErrAuthRequired indicates login/session is required.
	ErrAuthRequired = errors.New("telegram auth required")
	// ErrRetryable indicates a retryable transport failure.
	ErrRetryable = errors.New("retryable transport error")
)
