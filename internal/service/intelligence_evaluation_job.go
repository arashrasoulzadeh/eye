package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// RunIntelligenceEvaluationJob continuously evaluates the latest global and
// tenant behavior models. Evaluation is deliberately separate from training:
// a model can be retrained without becoming active, while its quality and
// drift status are still recorded for operators and automation.
func RunIntelligenceEvaluationJob(ctx context.Context, svc *UserIntelligenceService, interval time.Duration, logger *slog.Logger) {
	evaluate := func() {
		var models []model.IntelligenceModel
		if err := svc.DB.Where("label = ?", "behavior").Find(&models).Error; err != nil {
			if logger != nil {
				logger.Error("intelligence model scan failed", "err", err)
			}
			return
		}
		seen := map[string]bool{}
		for _, candidate := range models {
			key := candidate.TenantID + ":" + candidate.Label
			if seen[key] {
				continue
			}
			seen[key] = true
			if evaluation, err := svc.EvaluateLatest(candidate.TenantID, candidate.Label, time.Now()); err != nil && logger != nil {
				logger.Error("intelligence evaluation failed", "tenant", candidate.TenantID, "label", candidate.Label, "err", err)
			} else if err == nil {
				if alertErr := svc.RecordQualityAlert(evaluation); alertErr != nil && logger != nil {
					logger.Error("intelligence quality alert failed", "tenant", candidate.TenantID, "err", alertErr)
				}
			}
		}
	}
	evaluate()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			evaluate()
		}
	}
}
