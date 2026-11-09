package storage

import (
	"context"
	"log/slog"
	"time"
)

// RunRetentionJob periodically deletes events older than retention, until
// ctx is cancelled. Intended to be launched with `go RunRetentionJob(...)`.
//
// retentionFn is called fresh on every tick rather than taking a fixed
// time.Duration, so a live settings change (service.SettingsService,
// backing the admin /settings PUT endpoint) takes effect on the very next
// tick instead of requiring a process restart.
func RunRetentionJob(ctx context.Context, repo Repository, retentionFn func() time.Duration, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := repo.DeleteExpiredEvents(retentionFn())
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
