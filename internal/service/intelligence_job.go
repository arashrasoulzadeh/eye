package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

const intelligenceRetrainAfter = 6 * time.Hour

func intelligenceModelNeedsRetraining(model model.IntelligenceModel, now time.Time) bool {
	return model.ID == "" || model.CreatedAt.IsZero() || now.Sub(model.CreatedAt) >= intelligenceRetrainAfter
}

// RunUserIntelligenceJob refreshes every tenant's explainable snapshot on a
// schedule. A later supervised trainer can consume the same stored features.
func RunUserIntelligenceJob(ctx context.Context, svc *UserIntelligenceService, registry *ModelRegistry, interval time.Duration, logger *slog.Logger) {
	refresh := func() {
		var tenants []model.Tenant
		if err := svc.DB.Find(&tenants).Error; err != nil {
			if logger != nil {
				logger.Error("intelligence tenant scan failed", "err", err)
			}
			return
		}
		counts := make(map[string]int, len(tenants))
		for _, tenant := range tenants {
			if count, err := svc.Refresh(tenant.ID, time.Now()); err != nil {
				if logger != nil {
					logger.Error("intelligence refresh failed", "tenant", tenant.ID, "err", err)
				}
			} else {
				counts[tenant.ID] = count
			}
		}
		now := time.Now()
		var global model.IntelligenceModel
		globalReady := svc.DB.Where("tenant_id = ? AND label = ?", "", "behavior").Order("version desc").First(&global).Error == nil
		if !globalReady || intelligenceModelNeedsRetraining(global, now) {
			if _, err := svc.TrainUnsupervisedGlobal(now); err == nil {
				globalReady = true
			}
		}
		for _, tenant := range tenants {
			if _, ok := counts[tenant.ID]; !ok {
				continue
			}
			{
				var tenantModel model.IntelligenceModel
				hasTenantModel := svc.DB.Where("tenant_id = ? AND label = ?", tenant.ID, "behavior").Order("version desc").First(&tenantModel).Error == nil
				if globalReady && (!hasTenantModel || intelligenceModelNeedsRetraining(tenantModel, now)) {
					_, _ = svc.TrainUnsupervised(tenant.ID, now)
				}
				_ = svc.ApplyBehaviorLayers(tenant.ID)
				if registry != nil {
					if artifact, manifest, err := registry.Artifact("tenants/"+tenant.ID, "buyer"); err == nil {
						_ = svc.ApplyArtifact(tenant.ID, manifest.Task, artifact)
					} else if artifact, manifest, err := registry.Artifact("global", "buyer"); err == nil {
						_ = svc.ApplyArtifact(tenant.ID, manifest.Task, artifact)
					}
				}
				if logger != nil {
					logger.Debug("intelligence refreshed", "tenant", tenant.ID, "users", counts[tenant.ID])
				}
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
