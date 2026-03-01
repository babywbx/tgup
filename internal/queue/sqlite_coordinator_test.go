package queue

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestSQLiteCoordinatorFIFO(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "state.sqlite")
	c1 := openTestCoordinator(t, dbPath, "run-a")
	defer c1.Close()
	c2 := openTestCoordinator(t, dbPath, "run-b")
	defer c2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c1.WaitUntilTurn(ctx, nil); err != nil {
		t.Fatalf("c1 WaitUntilTurn() error = %v", err)
	}

	var waited atomic.Int64
	done := make(chan error, 1)
	go func() {
		done <- c2.WaitUntilTurn(ctx, func(ahead int) {
			if ahead > 0 {
				waited.Add(1)
			}
		})
	}()

	time.Sleep(200 * time.Millisecond)
	if err := c1.Finish(ctx, "finished"); err != nil {
		t.Fatalf("c1 Finish() error = %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("c2 WaitUntilTurn() error = %v", err)
	}
	if waited.Load() == 0 {
		t.Fatal("expected c2 to wait behind c1")
	}
}

func TestSQLiteCoordinatorRecoversStaleRunner(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "state.sqlite")
	opts := SQLiteOptions{
		HeartbeatTTL: 150 * time.Millisecond,
		PollInterval: 40 * time.Millisecond,
	}
	c1, err := OpenSQLite(dbPath, "run-a", opts)
	if err != nil {
		t.Fatalf("open c1: %v", err)
	}
	defer c1.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c1.WaitUntilTurn(ctx, nil); err != nil {
		t.Fatalf("c1 WaitUntilTurn() error = %v", err)
	}

	c2, err := OpenSQLite(dbPath, "run-b", opts)
	if err != nil {
		t.Fatalf("open c2: %v", err)
	}
	defer c2.Close()

	start := time.Now()
	if err := c2.WaitUntilTurn(ctx, nil); err != nil {
		t.Fatalf("c2 WaitUntilTurn() error = %v", err)
	}
	if time.Since(start) < opts.HeartbeatTTL {
		t.Fatalf("expected c2 to wait for stale timeout, waited %s", time.Since(start))
	}
}

func openTestCoordinator(t *testing.T, dbPath string, runID string) *SQLiteCoordinator {
	t.Helper()
	c, err := OpenSQLite(dbPath, runID, SQLiteOptions{
		HeartbeatTTL: time.Second,
		PollInterval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("OpenSQLite() error = %v", err)
	}
	return c
}
