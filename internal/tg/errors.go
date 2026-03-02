package tg

import (
	"errors"
	"time"
)

var (
	// ErrAuthRequired indicates login/session is required.
	ErrAuthRequired = errors.New("telegram auth required")
	// ErrRetryable indicates a retryable transport failure.
	ErrRetryable = errors.New("retryable transport error")
)

// FloodWaitError wraps a Telegram FLOOD_WAIT with the server-specified delay.
type FloodWaitError struct {
	Wait time.Duration
}

func (e *FloodWaitError) Error() string {
	return "FLOOD_WAIT_" + e.Wait.String()
}

// ImageProcessFailedError indicates Telegram rejected an image upload.
type ImageProcessFailedError struct{}

func (e *ImageProcessFailedError) Error() string {
	return "IMAGE_PROCESS_FAILED"
}

// IsImageProcessFailed reports whether err is an ImageProcessFailedError.
func IsImageProcessFailed(err error) bool {
	var target *ImageProcessFailedError
	return errors.As(err, &target)
}

// FloodWaitDuration returns the wait duration if err is a FloodWaitError,
// or 0 if not.
func FloodWaitDuration(err error) time.Duration {
	var target *FloodWaitError
	if errors.As(err, &target) {
		return target.Wait
	}
	return 0
}

// IsRetryable reports whether err is a retryable transport error.
// FloodWait and ImageProcessFailed are handled separately by the caller.
func IsRetryable(err error) bool {
	return errors.Is(err, ErrRetryable)
}
