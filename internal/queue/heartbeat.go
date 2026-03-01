package queue

import (
	"context"
	"time"
)

// StartHeartbeat periodically calls coordinator.Heartbeat until ctx ends.
func StartHeartbeat(ctx context.Context, coordinator Coordinator, interval time.Duration, onError func(error)) {
	if coordinator == nil || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := coordinator.Heartbeat(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}
