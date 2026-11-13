package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

const intelligenceRetrainAfter = 6 * time.Hour

func intelligenceModelNeedsRetraining(model model.IntelligenceModel, now time.Time) bool {
	return model.ID == "" || model.CreatedAt.IsZero() || now.Sub(model.CreatedAt) >= intelligenceRetrainAfter
}

func promoteHealthyUnsupervisedModel(svc *UserIntelligenceService, tenantID string, now time.Time, logger *slog.Logger) {
	var candidate model.IntelligenceModel
	if err := svc.DB.Where("tenant_id = ? AND label = ? AND status = ? AND algorithm = ?", tenantID, "behavior", "candidate", "unsupervised_behavior_v1").Order("version desc").First(&candidate).Error; err != nil {
		return
	}
	evaluation, err := svc.EvaluateLatest(tenantID, "behavior", now)
	if err != nil {
		if logger != nil {
			logger.Error("automatic intelligence evaluation failed", "tenant", tenantID, "err", err)
		}
		return
	}
	if evaluation.Status == "drifted" {
		return
	}
	if _, err := svc.PromoteLatest(tenantID, "behavior", now); err != nil && logger != nil {
		logger.Error("automatic intelligence promotion failed", "tenant", tenantID, "err", err)
	}
}

func trainEligibleSupervisedModels(svc *UserIntelligenceService, tenantID string, now time.Time, logger *slog.Logger) int {
	var labels []model.UserIntelligenceLabel
	if err := svc.DB.Where("tenant_id = ? AND (decision IS NULL OR decision <> ?)", tenantID, "denied").Find(&labels).Error; err != nil {
		if logger != nil {
			logger.Error("intelligence label scan failed", "tenant", tenantID, "err", err)
		}
		return 0
	}
	seen := map[string]bool{}
	trained := 0
	for _, labeled := range labels {
		if seen[labeled.Label] || labeled.Label == "behavior" {
			continue
		}
		seen[labeled.Label] = true
		readiness, err := svc.TrainingReadiness(tenantID, labeled.Label)
		if err != nil || readiness["ready"] != true {
			continue
		}
		var latest model.IntelligenceModel
		hasModel := svc.DB.Where("tenant_id = ? AND label = ?", tenantID, labeled.Label).Order("version desc").First(&latest).Error == nil
		if hasModel && !intelligenceModelNeedsRetraining(latest, now) {
			continue
		}
		candidate, err := svc.Train(tenantID, labeled.Label, now)
		if err != nil {
			if logger != nil {
				logger.Error("automatic supervised training failed", "tenant", tenantID, "label", labeled.Label, "err", err)
			}
			continue
		}
		evaluation, err := svc.EvaluateLatest(tenantID, labeled.Label, now)
		if err != nil || evaluation.Status == "drifted" {
			continue
		}
		if _, err := svc.PromoteLatest(tenantID, candidate.Label, now); err != nil && logger != nil {
			logger.Error("automatic supervised promotion failed", "tenant", tenantID, "label", candidate.Label, "err", err)
		} else {
			trained++
		}
	}
	return trained
}

// PipelineResult is the auditable result of a one-click intelligence run.
type PipelineResult struct {
	TenantID          string   `json:"tenant_id"`
	Refreshed         int      `json:"refreshed"`
	SupervisedTrained int      `json:"supervised_trained"`
	QualityStatus     string   `json:"quality_status"`
	Steps             []string `json:"steps"`
}

func (s *UserIntelligenceService) StartPipeline(tenantID string, now time.Time) (model.IntelligencePipelineRun, error) {
	run := model.IntelligencePipelineRun{ID: fmt.Sprintf("pipeline:%s:%d", tenantID, now.UnixNano()), TenantID: tenantID, Status: "queued", Progress: 0, Stage: "queued", CreatedAt: now}
	if err := s.DB.Create(&run).Error; err != nil {
		return run, err
	}
	go func() {
		started := time.Now()
		s.DB.Model(&model.IntelligencePipelineRun{}).Where("id = ?", run.ID).Updates(map[string]any{"status": "running", "stage": "running", "started_at": started})
		_, err := s.RunPipeline(tenantID, started)
		finished := time.Now()
		updates := map[string]any{"finished_at": finished}
		if err != nil {
			updates["status"] = "failed"
			updates["stage"] = "failed"
			updates["error"] = err.Error()
		} else {
			updates["status"] = "completed"
			updates["progress"] = 100
			updates["stage"] = "completed"
		}
		s.DB.Model(&model.IntelligencePipelineRun{}).Where("id = ?", run.ID).Updates(updates)
	}()
	return run, nil
}

func (s *UserIntelligenceService) PipelineRuns(tenantID string, limit int) ([]model.IntelligencePipelineRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var runs []model.IntelligencePipelineRun
	err := s.DB.Where("tenant_id = ?", tenantID).Order("created_at desc").Limit(limit).Find(&runs).Error
	return runs, err
}

// RunPipeline executes the complete synchronous intelligence lifecycle for one
// tenant. The scheduled job uses the same individual operations, while this
// method gives the admin a single auditable action with explicit stages.
func (s *UserIntelligenceService) RunPipeline(tenantID string, now time.Time) (PipelineResult, error) {
	result := PipelineResult{TenantID: tenantID, Steps: make([]string, 0, 9)}
	refreshed, err := s.Refresh(tenantID, now)
	if err != nil {
		return result, fmt.Errorf("refresh: %w", err)
	}
	result.Refreshed = refreshed
	result.Steps = append(result.Steps, "refresh")
	if _, err = s.TrainUnsupervisedGlobal(now); err != nil {
		return result, fmt.Errorf("global training: %w", err)
	}
	result.Steps = append(result.Steps, "global_train")
	if evaluation, evalErr := s.EvaluateLatest("", "behavior", now); evalErr != nil {
		return result, fmt.Errorf("global evaluation: %w", evalErr)
	} else if evaluation.Status == "drifted" {
		return result, fmt.Errorf("global evaluation drifted")
	}
	result.Steps = append(result.Steps, "global_evaluate")
	if _, err = s.PromoteLatest("", "behavior", now); err != nil {
		return result, fmt.Errorf("global promotion: %w", err)
	}
	result.Steps = append(result.Steps, "global_promote")
	if _, err = s.TrainUnsupervised(tenantID, now); err != nil {
		return result, fmt.Errorf("tenant training: %w", err)
	}
	result.Steps = append(result.Steps, "tenant_train")
	if evaluation, evalErr := s.EvaluateLatest(tenantID, "behavior", now); evalErr != nil {
		return result, fmt.Errorf("tenant evaluation: %w", evalErr)
	} else if evaluation.Status == "drifted" {
		return result, fmt.Errorf("tenant evaluation drifted")
	}
	result.Steps = append(result.Steps, "tenant_evaluate")
	if _, err = s.PromoteLatest(tenantID, "behavior", now); err != nil {
		return result, fmt.Errorf("tenant promotion: %w", err)
	}
	result.Steps = append(result.Steps, "tenant_promote")
	result.SupervisedTrained = trainEligibleSupervisedModels(s, tenantID, now, nil)
	result.Steps = append(result.Steps, "supervised_train")
	if quality, qualityErr := s.AssessDataQuality(tenantID, now); qualityErr != nil {
		return result, fmt.Errorf("quality assessment: %w", qualityErr)
	} else {
		result.QualityStatus = quality.Status
	}
	result.Steps = append(result.Steps, "quality_assess")
	if err = s.ApplyBehaviorLayers(tenantID); err != nil {
		return result, fmt.Errorf("apply behavior layers: %w", err)
	}
	result.Steps = append(result.Steps, "apply_layers")
	return result, nil
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
			} else if logger != nil {
				logger.Error("automatic global intelligence training failed", "err", err)
			}
		}
		promoteHealthyUnsupervisedModel(svc, "", now, logger)
		for _, tenant := range tenants {
			if _, ok := counts[tenant.ID]; !ok {
				continue
			}
			{
				var tenantModel model.IntelligenceModel
				hasTenantModel := svc.DB.Where("tenant_id = ? AND label = ?", tenant.ID, "behavior").Order("version desc").First(&tenantModel).Error == nil
				if globalReady && (!hasTenantModel || intelligenceModelNeedsRetraining(tenantModel, now)) {
					if _, err := svc.TrainUnsupervised(tenant.ID, now); err != nil && logger != nil {
						logger.Error("automatic intelligence training failed", "tenant", tenant.ID, "err", err)
					}
				}
				promoteHealthyUnsupervisedModel(svc, tenant.ID, now, logger)
				trainEligibleSupervisedModels(svc, tenant.ID, now, logger)
				if err := svc.ApplyBehaviorLayers(tenant.ID); err != nil && logger != nil {
					logger.Error("intelligence layer application failed", "tenant", tenant.ID, "err", err)
				}
				if registry != nil {
					if artifact, manifest, err := registry.Artifact("tenants/"+tenant.ID, "buyer"); err == nil {
						if applyErr := svc.ApplyArtifact(tenant.ID, manifest.Task, artifact); applyErr != nil && logger != nil {
							logger.Error("tenant intelligence artifact application failed", "tenant", tenant.ID, "err", applyErr)
						}
					} else if artifact, manifest, err := registry.Artifact("global", "buyer"); err == nil {
						if applyErr := svc.ApplyArtifact(tenant.ID, manifest.Task, artifact); applyErr != nil && logger != nil {
							logger.Error("global intelligence artifact application failed", "tenant", tenant.ID, "err", applyErr)
						}
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
