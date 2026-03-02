package tg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/gotd/td/tgerr"
)

// mapGotdError converts a gotd error into our app-level error types.
func mapGotdError(err error) error {
	if err == nil {
		return nil
	}

	// Pass through context errors directly — callers check these.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	// FloodWait → our FloodWaitError.
	if dur, ok := tgerr.AsFloodWait(err); ok {
		return &FloodWaitError{Wait: dur}
	}

	// Photo-rejected errors → our ImageProcessFailedError.
	// Triggers auto-fallback to document mode in sendWithRetry.
	if tgerr.Is(err, "IMAGE_PROCESS_FAILED") || tgerr.Is(err, "PHOTO_SAVE_FILE_INVALID") {
		return &ImageProcessFailedError{}
	}

	// RPC errors.
	if rpcErr, ok := tgerr.As(err); ok {
		switch {
		case rpcErr.Code == 401:
			return fmt.Errorf("%s: %w", rpcErr.Message, ErrAuthRequired)
		case rpcErr.Code >= 500:
			return fmt.Errorf("%s: %w", rpcErr.Message, ErrRetryable)
		default:
			return fmt.Errorf("telegram: %s (code %d)", rpcErr.Message, rpcErr.Code)
		}
	}

	// Network-level errors are retryable.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Errorf("%v: %w", err, ErrRetryable)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("%v: %w", err, ErrRetryable)
	}

	return err
}
