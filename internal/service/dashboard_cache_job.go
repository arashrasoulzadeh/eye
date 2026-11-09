package service

import (
	"context"
	"log/slog"
	"time"
)

// RunDashboardCacheJob periodically refreshes the dashboard cache.
func RunDashboardCacheJob(ctx context.Context, svc *DashboardCacheService, interval time.Duration, logger *slog.Logger) {
	// Initial refresh
	if err := svc.Refresh(); err != nil {
		logger.Error("initial dashboard cache refresh failed", "err", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.Refresh(); err != nil {
				logger.Error("dashboard cache refresh failed", "err", err)
			}
		}
	}
}