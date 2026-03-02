package tg

import (
	"context"
	"errors"
	"io"
	"math"
	"math/rand"
	"net"
	"time"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// --- floodwait middleware ---

// floodWaitMiddleware retries RPC calls that hit FLOOD_WAIT by sleeping
// the exact duration requested by the server.
type floodWaitMiddleware struct {
	maxRetries int
	maxWait    time.Duration
}

// NewFloodWaitMiddleware creates a middleware that auto-waits on FLOOD_WAIT.
func NewFloodWaitMiddleware(maxRetries int, maxWait time.Duration) telegram.Middleware {
	return &floodWaitMiddleware{maxRetries: maxRetries, maxWait: maxWait}
}

func (m *floodWaitMiddleware) Handle(next tg.Invoker) telegram.InvokeFunc {
	return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
		for attempt := 0; ; attempt++ {
			err := next.Invoke(ctx, input, output)
			if err == nil {
				return nil
			}

			d, ok := tgerr.AsFloodWait(err)
			if !ok {
				return err
			}

			if attempt >= m.maxRetries {
				return err
			}
			// Server sometimes returns 0; use 1s floor.
			if d <= 0 {
				d = time.Second
			}
			if m.maxWait > 0 && d > m.maxWait {
				return err
			}

			timer := time.NewTimer(d)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
}

// --- retry middleware ---

// transientTypes are Telegram-internal RPC error type strings that indicate
// transient server-side issues worth retrying immediately.
var transientTypes = []string{
	"Timedout",
	"RPC_CALL_FAIL",
	"RPC_MCGET_FAIL",
	"WORKER_BUSY_TOO_LONG_RETRY",
	"No workers running",
	// 400-level but documented as "Internal issues, try again later".
	"PHOTO_SAVE_FILE_INVALID",
}

// retryMiddleware retries on Telegram transient server errors without delay.
type retryMiddleware struct {
	maxRetries int
}

// NewRetryMiddleware creates a middleware that immediately retries on
// transient Telegram server errors (code >= 500 or known error types).
func NewRetryMiddleware(maxRetries int) telegram.Middleware {
	return &retryMiddleware{maxRetries: maxRetries}
}

func (m *retryMiddleware) Handle(next tg.Invoker) telegram.InvokeFunc {
	return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
		for attempt := 0; ; attempt++ {
			err := next.Invoke(ctx, input, output)
			if err == nil {
				return nil
			}
			if ctx.Err() != nil {
				return err
			}
			if attempt >= m.maxRetries {
				return err
			}
			if !isTransientRPC(err) {
				return err
			}
		}
	}
}

// isTransientRPC reports whether err is a transient Telegram RPC error.
func isTransientRPC(err error) bool {
	rpcErr, ok := tgerr.As(err)
	if !ok {
		return false
	}
	if rpcErr.Code >= 500 {
		return true
	}
	return rpcErr.IsOneOf(transientTypes...)
}

// --- recovery middleware ---

// recoveryMiddleware retries on non-RPC errors (network, IO) with
// exponential backoff. RPC errors pass through immediately.
type recoveryMiddleware struct {
	maxElapsed time.Duration
	initDelay  time.Duration
	maxDelay   time.Duration
	multiplier float64
}

// NewRecoveryMiddleware creates a middleware that recovers from transport-level
// errors (network drops, IO errors) using exponential backoff.
func NewRecoveryMiddleware(maxElapsed time.Duration) telegram.Middleware {
	return &recoveryMiddleware{
		maxElapsed: maxElapsed,
		initDelay:  time.Second,
		maxDelay:   10 * time.Second,
		multiplier: 2.0,
	}
}

func (m *recoveryMiddleware) Handle(next tg.Invoker) telegram.InvokeFunc {
	return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
		start := time.Now()
		delay := m.initDelay

		for {
			err := next.Invoke(ctx, input, output)
			if err == nil {
				return nil
			}
			if ctx.Err() != nil {
				return err
			}
			if !isTransportError(err) {
				return err
			}
			if time.Since(start) >= m.maxElapsed {
				return err
			}

			// Jitter: 0.5x ~ 1.0x of current delay.
			jittered := time.Duration(float64(delay) * (0.5 + rand.Float64()*0.5))
			timer := time.NewTimer(jittered)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}

			delay = time.Duration(math.Min(
				float64(delay)*m.multiplier,
				float64(m.maxDelay),
			))
		}
	}
}

// isTransportError reports whether err is a non-RPC transport/network error
// that should be recovered with backoff.
func isTransportError(err error) bool {
	// RPC errors are business-level — do not recover.
	if _, ok := tgerr.As(err); ok {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	return false
}
