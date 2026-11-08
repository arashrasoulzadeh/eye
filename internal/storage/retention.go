package storage

import (
	"context"
	"log/slog"
	"time"
)

// RunRetentionJob periodically deletes events older than retention, until
// ctx is cancelled. Intended to be launched with `go RunRetentionJob(...)`.
func RunRetentionJob(ctx context.Context, repo Repository, retention, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := repo.DeleteExpiredEvents(retention)
			if err != nil {
				if logger != nil {
					logger.Error("retention job failed", "err", err)
				}
				continue
			}
			if logger != nil && n > 0 {
				logger.Info("retention job deleted expired events", "count", n)
			}
		}
	}
}
