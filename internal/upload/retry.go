package upload

import "time"

// RetryDecision describes how upload should retry on failure.
type RetryDecision struct {
	Retry bool
	After time.Duration
}

// Backoff computes an exponential-like retry delay with an upper bound.
func Backoff(attempt int) time.Duration {
	if attempt <= 0 {
		return time.Second
	}
	delay := time.Second << (attempt - 1)
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}
