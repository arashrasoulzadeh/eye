package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// RunUserIntelligenceJob refreshes every tenant's explainable snapshot on a
// schedule. A later supervised trainer can consume the same stored features.
func RunUserIntelligenceJob(ctx context.Context, svc *UserIntelligenceService, interval time.Duration, logger *slog.Logger) {
	refresh := func() {
		var tenants []model.Tenant
		if err := svc.DB.Find(&tenants).Error; err != nil {
			if logger != nil {
				logger.Error("intelligence tenant scan failed", "err", err)
			}
			return
		}
		for _, tenant := range tenants {
			if count, err := svc.Refresh(tenant.ID, time.Now()); err != nil {
				if logger != nil {
					logger.Error("intelligence refresh failed", "tenant", tenant.ID, "err", err)
				}
			} else if logger != nil {
				logger.Debug("intelligence refreshed", "tenant", tenant.ID, "users", count)
			}
		}
	}
	refresh()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresh()
		}
	}
}
