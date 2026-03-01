package state

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStoreMarkAndResume(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer store.Close()

	ctx := context.Background()
	sentKey := ResumeKey{Path: "/tmp/a.jpg", Size: 10, MTimeNS: 100}
	failedKey := ResumeKey{Path: "/tmp/b.jpg", Size: 20, MTimeNS: 200}

	if err := store.MarkSent(ctx, MarkSentInput{
		Key:          sentKey,
		Target:       "me",
		MessageIDs:   []int{1, 2},
		AlbumGroupID: "grp-1",
	}); err != nil {
		t.Fatalf("MarkSent() error = %v", err)
	}
	if err := store.MarkFailed(ctx, MarkFailedInput{
		Key:         failedKey,
		Target:      "me",
		ErrorReason: "network",
	}); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}

	done, err := store.IsDone(ctx, sentKey)
	if err != nil {
		t.Fatalf("IsDone(sent) error = %v", err)
	}
	if !done {
		t.Fatal("expected sent key done")
	}

	done, err = store.IsDone(ctx, failedKey)
	if err != nil {
		t.Fatalf("IsDone(failed) error = %v", err)
	}
	if done {
		t.Fatal("expected failed key not done")
	}

	row, err := store.GetUploadRow(ctx, sentKey)
	if err != nil {
		t.Fatalf("GetUploadRow() error = %v", err)
	}
	if row == nil || row.Status != "sent" || row.AlbumGroupID != "grp-1" || len(row.MessageIDs) != 2 {
		t.Fatalf("unexpected sent row: %#v", row)
	}

	pending, err := store.ListPending(ctx, []ResumeKey{sentKey, failedKey})
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(pending) != 1 || pending[0].Path != failedKey.Path {
		t.Fatalf("unexpected pending: %#v", pending)
	}
}

func TestSQLiteStoreApplyMaintenance(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer store.Close()
	ctx := context.Background()

	oldKey := ResumeKey{Path: "/tmp/old.jpg", Size: 1, MTimeNS: 1}
	if err := store.MarkFailed(ctx, MarkFailedInput{
		Key:         oldKey,
		Target:      "me",
		ErrorReason: "x",
	}); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE uploads SET updated_at = ? WHERE path = ? AND size = ? AND mtime_ns = ?
	`, time.Now().Add(-2*time.Hour).Unix(), oldKey.Path, oldKey.Size, oldKey.MTimeNS); err != nil {
		t.Fatalf("update uploads updated_at: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO run_queue(run_id, status, created_at, heartbeat_at, finished_at) VALUES(?, 'finished', ?, ?, ?)
	`, "run-1", time.Now().Add(-2*time.Hour).Unix(), time.Now().Add(-2*time.Hour).Unix(), time.Now().Add(-2*time.Hour).Unix()); err != nil {
		t.Fatalf("insert run_queue row: %v", err)
	}

	preview, err := store.ApplyMaintenance(ctx, MaintenanceConfig{
		Enabled: true,
		MaxAge:  time.Hour,
	}, false)
	if err != nil {
		t.Fatalf("ApplyMaintenance(preview) error = %v", err)
	}
	if !preview.Preview || preview.DeletedUploads == 0 || preview.DeletedQueueRows == 0 {
		t.Fatalf("unexpected preview report: %#v", preview)
	}

	applied, err := store.ApplyMaintenance(ctx, MaintenanceConfig{
		Enabled: true,
		MaxAge:  time.Hour,
	}, true)
	if err != nil {
		t.Fatalf("ApplyMaintenance(apply) error = %v", err)
	}
	if applied.Preview {
		t.Fatalf("expected apply report, got %#v", applied)
	}

	row, err := store.GetUploadRow(ctx, oldKey)
	if err != nil {
		t.Fatalf("GetUploadRow(old) error = %v", err)
	}
	if row != nil {
		t.Fatalf("expected old row deleted, got %#v", row)
	}
}

func openTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.sqlite")
	store, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("OpenSQLite() error = %v", err)
	}
	return store
}
