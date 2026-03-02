package tg

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/pool"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// mockInvoker counts calls and returns a sequence of errors.
type mockInvoker struct {
	calls  atomic.Int32
	errors []error
}

func (m *mockInvoker) Invoke(_ context.Context, _ bin.Encoder, _ bin.Decoder) error {
	i := int(m.calls.Add(1)) - 1
	if i >= len(m.errors) {
		return nil
	}
	return m.errors[i]
}

func newMock(errs ...error) *mockInvoker {
	return &mockInvoker{errors: errs}
}

// --- floodwait tests ---

func TestFloodWait_WaitsAndRetries(t *testing.T) {
	floodErr := tgerr.New(420, "FLOOD_WAIT_1")
	mock := newMock(floodErr) // first call floods, second succeeds
	mw := NewFloodWaitMiddleware(5, time.Minute)
	fn := mw.Handle(mock)

	start := time.Now()
	err := fn.Invoke(context.Background(), nil, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if mock.calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", mock.calls.Load())
	}
	if elapsed < 900*time.Millisecond {
		t.Fatalf("expected >=900ms wait, got %v", elapsed)
	}
}

func TestFloodWait_MaxRetriesExceeded(t *testing.T) {
	floodErr := tgerr.New(420, "FLOOD_WAIT_0")
	mock := newMock(floodErr, floodErr, floodErr)
	mw := NewFloodWaitMiddleware(2, time.Minute)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	// 1 initial + 2 retries = 3 calls, then gives up
	if mock.calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", mock.calls.Load())
	}
}

func TestFloodWait_ExceedsMaxWait(t *testing.T) {
	// FLOOD_WAIT_3600 (1 hour) exceeds maxWait of 1 minute
	floodErr := tgerr.New(420, "FLOOD_WAIT_3600")
	mock := newMock(floodErr)
	mw := NewFloodWaitMiddleware(5, time.Minute)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for excessive wait")
	}
	if mock.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls.Load())
	}
}

func TestFloodWait_NonFloodPassesThrough(t *testing.T) {
	rpcErr := tgerr.New(400, "PEER_ID_INVALID")
	mock := newMock(rpcErr)
	mw := NewFloodWaitMiddleware(5, time.Minute)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if mock.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls.Load())
	}
}

func TestFloodWait_ContextCanceled(t *testing.T) {
	floodErr := tgerr.New(420, "FLOOD_WAIT_60")
	mock := newMock(floodErr)
	mw := NewFloodWaitMiddleware(5, 2*time.Minute)
	fn := mw.Handle(mock)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately after first call returns flood error.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := fn.Invoke(ctx, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// --- retry tests ---

func TestRetry_TransientServerError(t *testing.T) {
	serverErr := tgerr.New(500, "INTERNAL")
	mock := newMock(serverErr) // first fails, second succeeds
	mw := NewRetryMiddleware(5)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if mock.calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", mock.calls.Load())
	}
}

func TestRetry_KnownTransientType(t *testing.T) {
	for _, typ := range transientTypes {
		t.Run(typ, func(t *testing.T) {
			rpcErr := tgerr.New(400, typ)
			mock := newMock(rpcErr) // first fails, second succeeds
			mw := NewRetryMiddleware(3)
			fn := mw.Handle(mock)

			err := fn.Invoke(context.Background(), nil, nil)
			if err != nil {
				t.Fatalf("expected nil after retry, got %v", err)
			}
			if mock.calls.Load() != 2 {
				t.Fatalf("expected 2 calls, got %d", mock.calls.Load())
			}
		})
	}
}

func TestRetry_MaxRetriesExceeded(t *testing.T) {
	serverErr := tgerr.New(500, "INTERNAL")
	mock := newMock(serverErr, serverErr, serverErr, serverErr)
	mw := NewRetryMiddleware(2)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	// 1 initial + 2 retries = 3 calls
	if mock.calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", mock.calls.Load())
	}
}

func TestRetry_NonTransientPassesThrough(t *testing.T) {
	rpcErr := tgerr.New(400, "PEER_ID_INVALID")
	mock := newMock(rpcErr)
	mw := NewRetryMiddleware(5)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if mock.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls.Load())
	}
}

func TestRetry_NetworkErrorNotHandled(t *testing.T) {
	// retry middleware only handles RPC errors; network errors are for recovery.
	netErr := &net.OpError{Op: "read", Err: errors.New("reset")}
	mock := newMock(netErr)
	mw := NewRetryMiddleware(5)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if mock.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls.Load())
	}
}

// --- recovery tests ---

func TestRecovery_NetworkErrorRecovered(t *testing.T) {
	netErr := &net.OpError{Op: "read", Err: errors.New("connection reset")}
	mock := newMock(netErr) // first fails, second succeeds
	mw := NewRecoveryMiddleware(30 * time.Second)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if mock.calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", mock.calls.Load())
	}
}

func TestRecovery_EOFRecovered(t *testing.T) {
	mock := newMock(io.EOF) // first fails, second succeeds
	mw := NewRecoveryMiddleware(30 * time.Second)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if mock.calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", mock.calls.Load())
	}
}

func TestRecovery_UnexpectedEOFRecovered(t *testing.T) {
	mock := newMock(io.ErrUnexpectedEOF)
	mw := NewRecoveryMiddleware(30 * time.Second)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if mock.calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", mock.calls.Load())
	}
}

func TestRecovery_ConnDeadRecovered(t *testing.T) {
	mock := newMock(pool.ErrConnDead) // first fails, second succeeds
	mw := NewRecoveryMiddleware(30 * time.Second)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if mock.calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", mock.calls.Load())
	}
}

func TestRecovery_RPCErrorPassesThrough(t *testing.T) {
	rpcErr := tgerr.New(400, "PEER_ID_INVALID")
	mock := newMock(rpcErr)
	mw := NewRecoveryMiddleware(30 * time.Second)
	fn := mw.Handle(mock)

	err := fn.Invoke(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if mock.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls.Load())
	}
}

func TestRecovery_ContextCanceled(t *testing.T) {
	// Keep returning network errors forever.
	netErr := &net.OpError{Op: "read", Err: errors.New("reset")}
	mock := newMock(netErr, netErr, netErr, netErr, netErr, netErr)
	mw := NewRecoveryMiddleware(30 * time.Second)
	fn := mw.Handle(mock)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := fn.Invoke(ctx, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// --- isTransientRPC tests ---

func TestIsTransientRPC(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"500 error", tgerr.New(500, "INTERNAL"), true},
		{"timedout", tgerr.New(400, "Timedout"), true},
		{"rpc_call_fail", tgerr.New(400, "RPC_CALL_FAIL"), true},
		{"normal rpc error", tgerr.New(400, "PEER_ID_INVALID"), false},
		{"network error", &net.OpError{Op: "read", Err: errors.New("reset")}, false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTransientRPC(tt.err)
			if got != tt.expect {
				t.Fatalf("isTransientRPC() = %v, want %v", got, tt.expect)
			}
		})
	}
}

// --- isTransportError tests ---

func TestIsTransportError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"net error", &net.OpError{Op: "read", Err: errors.New("reset")}, true},
		{"io.EOF", io.EOF, true},
		{"unexpected eof", io.ErrUnexpectedEOF, true},
		{"conn dead", pool.ErrConnDead, true},
		{"rpc error", tgerr.New(400, "BAD_REQUEST"), false},
		{"plain error", errors.New("something"), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTransportError(tt.err)
			if got != tt.expect {
				t.Fatalf("isTransportError() = %v, want %v", got, tt.expect)
			}
		})
	}
}

// --- integration: middleware chain order ---

func TestMiddlewareChain_Order(t *testing.T) {
	// Verify that the chain processes: recovery -> retry -> floodwait -> invoke.
	var order []string

	// Build a chain that records execution order.
	recovery := middlewareFunc(func(next tg.Invoker) func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
		return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
			order = append(order, "recovery-before")
			err := next.Invoke(ctx, input, output)
			order = append(order, "recovery-after")
			return err
		}
	})
	retry := middlewareFunc(func(next tg.Invoker) func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
		return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
			order = append(order, "retry-before")
			err := next.Invoke(ctx, input, output)
			order = append(order, "retry-after")
			return err
		}
	})
	flood := middlewareFunc(func(next tg.Invoker) func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
		return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
			order = append(order, "flood-before")
			err := next.Invoke(ctx, input, output)
			order = append(order, "flood-after")
			return err
		}
	})

	mock := newMock() // succeeds immediately
	// Chain manually in the same order as gotd: first middleware is outermost.
	var invoker tg.Invoker = mock
	for _, mw := range []middlewareFunc{flood, retry, recovery} {
		invoker = mw.handle(invoker)
	}

	err := invoker.Invoke(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"recovery-before", "retry-before", "flood-before",
		"flood-after", "retry-after", "recovery-after",
	}
	if len(order) != len(expected) {
		t.Fatalf("order mismatch: got %v, want %v", order, expected)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], expected[i])
		}
	}
}

type middlewareFunc func(next tg.Invoker) func(ctx context.Context, input bin.Encoder, output bin.Decoder) error

func (m middlewareFunc) handle(next tg.Invoker) tg.Invoker {
	fn := m(next)
	return invokerFunc(fn)
}

type invokerFunc func(ctx context.Context, input bin.Encoder, output bin.Decoder) error

func (f invokerFunc) Invoke(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
	return f(ctx, input, output)
}
