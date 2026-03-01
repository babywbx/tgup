package state

func buildUploadCleanupWhere(cfg MaintenanceConfig, cutoffUnix int64) (string, []any) {
	var where string
	var args []any
	if cfg.KeepFailed {
		where = "status = ?"
		args = []any{"failed"}
	} else {
		where = "status IN (?, ?)"
		args = []any{"failed", "sent"}
	}
	if cutoffUnix > 0 {
		where += " AND updated_at < ?"
		args = append(args, cutoffUnix)
	}
	return where, args
}

func buildQueueCleanupWhere(cutoffUnix int64) (string, []any) {
	where := "status IN (?, ?, ?, ?)"
	args := []any{"finished", "failed", "canceled", "stale"}
	if cutoffUnix > 0 {
		where += " AND COALESCE(finished_at, 0) < ?"
		args = append(args, cutoffUnix)
	}
	return where, args
}
