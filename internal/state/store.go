package state

import "context"

// Store defines app-owned state persistence operations.
type Store interface {
	MarkSent(ctx context.Context, in MarkSentInput) error
	MarkFailed(ctx context.Context, in MarkFailedInput) error
	IsDone(ctx context.Context, item ResumeKey) (bool, error)
	ListPending(ctx context.Context, items []ResumeKey) ([]ResumeKey, error)
	GetUploadRow(ctx context.Context, item ResumeKey) (*UploadRow, error)
	ApplyMaintenance(ctx context.Context, cfg MaintenanceConfig, force bool) (CleanupReport, error)
	Close() error
}
