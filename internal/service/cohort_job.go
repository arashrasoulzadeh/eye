package service

import (
	"context"
	"log/slog"
	"time"
)

// RunCohortMaterializationJob periodically re-materializes every cohort
// (across all tenants) until ctx is cancelled. This is a simple on-demand
// substitute for a real stream processor (Phase 7/Kafka isn't built yet):
// each tick does a full recompute rather than an incremental update.
// Intended to be launched with `go RunCohortMaterializationJob(...)`.
func RunCohortMaterializationJob(ctx context.Context, svc *CohortService, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var ids []string
			if err := svc.DB.Table("cohorts").Pluck("id", &ids).Error; err != nil {
				if logger != nil {
					logger.Error("cohort materialization job: list cohorts failed", "err", err)
				}
				continue
			}
			for _, id := range ids {
				if _, err := svc.Materialize(id); err != nil && logger != nil {
					logger.Error("cohort materialization failed", "cohort_id", id, "err", err)
				}
			}
		}
	}
}
