package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/metrics"
	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
)

type UserIntelligenceService struct{ DB *gorm.DB }

func (s *UserIntelligenceService) AssessDataQuality(tenantID string, now time.Time) (model.IntelligenceDataQuality, error) {
	quality := model.IntelligenceDataQuality{ID: fmt.Sprintf("%s:%d", tenantID, now.UnixNano()), TenantID: tenantID, CreatedAt: now}
	var profiles []model.UserIntelligence
	query := s.DB
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Find(&profiles).Error; err != nil {
		return quality, err
	}
	quality.Profiles = len(profiles)
	seen := map[string]bool{}
	for _, profile := range profiles {
		if profile.IP == "" && profile.UserAgent == "" {
			quality.MissingIdentity++
		}
		if strings.TrimSpace(profile.Features) == "" || profile.Features == "{}" || profile.Features == "null" {
			quality.MissingFeatures++
		}
		if seen[profile.Fingerprint] {
			quality.DuplicateUsers++
		}
		seen[profile.Fingerprint] = true
	}
	eventQuery := s.DB.Where("is_aggregated = ?", false)
	sessionQuery := s.DB
	if tenantID != "" {
		eventQuery = eventQuery.Where("tenant_id = ?", tenantID)
		sessionQuery = sessionQuery.Where("tenant_id = ?", tenantID)
	}
	var events []model.Event
	var sessions []model.Session
	if err := eventQuery.Find(&events).Error; err != nil {
		return quality, err
	}
	if err := sessionQuery.Find(&sessions).Error; err != nil {
		return quality, err
	}
	quality.Events, quality.Sessions = len(events), len(sessions)
	denominator := float64(quality.Profiles * 2)
	missing := float64(quality.MissingIdentity + quality.MissingFeatures)
	quality.CoverageScore = 1
	if denominator > 0 {
		quality.CoverageScore = math.Max(0, 1-missing/denominator)
	}
	quality.Status = "healthy"
	if quality.CoverageScore < 0.7 {
		quality.Status = "poor"
	} else if quality.CoverageScore < 0.9 {
		quality.Status = "warning"
	}
	details, _ := json.Marshal(map[string]any{"identity_coverage": 1 - float64(quality.MissingIdentity)/math.Max(1, float64(quality.Profiles)), "feature_coverage": 1 - float64(quality.MissingFeatures)/math.Max(1, float64(quality.Profiles)), "checked_at": now})
	quality.Details = string(details)
	if err := s.DB.Create(&quality).Error; err != nil {
		return quality, err
	}
	return quality, nil
}

func (s *UserIntelligenceService) LatestDataQuality(tenantID string) (model.IntelligenceDataQuality, error) {
	var quality model.IntelligenceDataQuality
	err := s.DB.Where("tenant_id = ?", tenantID).Order("created_at desc").First(&quality).Error
	return quality, err
}

func (s *UserIntelligenceService) RecordQualityAlert(evaluation model.IntelligenceEvaluation) error {
	kind, severity, message := "", "warning", ""
	if evaluation.DriftScore >= 0.3 {
		kind, severity, message = "model_drift", "critical", fmt.Sprintf("model drift score %.3f exceeds threshold", evaluation.DriftScore)
	} else if evaluation.CalibrationError >= 0.2 {
		kind, severity, message = "calibration", "warning", fmt.Sprintf("calibration error %.3f exceeds threshold", evaluation.CalibrationError)
	}
	if kind == "" {
		return nil
	}
	var open model.IntelligenceAlert
	err := s.DB.Where("tenant_id = ? AND label = ? AND kind = ? AND status = ?", evaluation.TenantID, evaluation.Label, kind, "open").First(&open).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	alert := model.IntelligenceAlert{ID: fmt.Sprintf("%s:%s:%s:%d", evaluation.TenantID, evaluation.Label, kind, evaluation.CreatedAt.UnixNano()), TenantID: evaluation.TenantID, Label: evaluation.Label, Kind: kind, Severity: severity, Status: "open", Message: message, CreatedAt: evaluation.CreatedAt}
	return s.DB.Create(&alert).Error
}

func (s *UserIntelligenceService) IntelligenceAlerts(tenantID, status string, limit int) ([]model.IntelligenceAlert, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := s.DB.Where("tenant_id = ?", tenantID).Order("created_at desc").Limit(limit)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var alerts []model.IntelligenceAlert
	return alerts, query.Find(&alerts).Error
}

func (s *UserIntelligenceService) ResolveIntelligenceAlert(tenantID, id string, now time.Time) error {
	result := s.DB.Model(&model.IntelligenceAlert{}).Where("id = ? AND tenant_id = ? AND status = ?", id, tenantID, "open").Updates(map[string]any{"status": "resolved", "resolved_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *UserIntelligenceService) EvaluateLatest(tenantID, label string, now time.Time) (model.IntelligenceEvaluation, error) {
	var trained model.IntelligenceModel
	query := s.DB.Where("tenant_id = ? AND label = ?", tenantID, label).Order("version desc")
	if err := query.First(&trained).Error; err != nil {
		return model.IntelligenceEvaluation{}, err
	}
	var report map[string]any
	_ = json.Unmarshal([]byte(trained.Metrics), &report)
	evaluation := model.IntelligenceEvaluation{ID: fmt.Sprintf("%s:%s:%d:%d", tenantID, label, trained.Version, now.UnixNano()), TenantID: tenantID, Label: label, ModelVersion: trained.Version, Status: "healthy", CreatedAt: now}
	for key, target := range []*float64{&evaluation.Accuracy, &evaluation.Precision, &evaluation.Recall, &evaluation.F1} {
		if value, ok := report[[]string{"accuracy", "precision", "recall", "f1"}[key]].(float64); ok {
			*target = value
		}
	}
	if label != "behavior" {
		var labels []model.UserIntelligenceLabel
		if err := s.DB.Where("tenant_id = ? AND (decision IS NULL OR decision <> ?)", tenantID, "denied").Find(&labels).Error; err != nil {
			return evaluation, err
		}
		if len(labels) > 0 {
			var brier float64
			bins := make([][]float64, 10)
			for _, labeled := range labels {
				var snapshot model.UserIntelligence
				if s.DB.Where("tenant_id = ? AND fingerprint = ?", tenantID, labeled.Fingerprint).First(&snapshot).Error != nil {
					continue
				}
				var scores map[string]float64
				_ = json.Unmarshal([]byte(snapshot.Scores), &scores)
				probability := math.Max(0, math.Min(1, scores[label]))
				actual := 0.0
				if labeled.Label == label {
					actual = 1
				}
				brier += (probability - actual) * (probability - actual)
				bin := int(probability * 10)
				if bin == 10 {
					bin = 9
				}
				bins[bin] = append(bins[bin], probability)
			}
			evaluation.BrierScore = brier / float64(len(labels))
			for _, values := range bins {
				if len(values) > 0 {
					var sum float64
					for _, value := range values {
						sum += value
					}
					evaluation.CalibrationError += math.Abs(sum/float64(len(values))-0.5) * float64(len(values)) / float64(len(labels))
				}
			}
		}
	}
	if label == "behavior" {
		var rows []model.UserIntelligence
		if err := s.DB.Where("tenant_id = ?", tenantID).Find(&rows).Error; err != nil {
			return evaluation, err
		}
		current := map[string]float64{}
		for _, row := range rows {
			var tags []string
			_ = json.Unmarshal([]byte(row.Tags), &tags)
			for _, tag := range tags {
				current[tag]++
			}
		}
		var baseline map[string]any
		_ = json.Unmarshal([]byte(trained.Metrics), &baseline)
		segments, _ := baseline["segments"].(map[string]any)
		if len(rows) > 0 && len(segments) > 0 {
			var drift float64
			for tag, count := range current {
				expected, _ := segments[tag].(float64)
				drift += math.Abs(count/float64(len(rows)) - expected/float64(len(rows)))
			}
			evaluation.DriftScore = math.Min(1, drift)
		}
	}
	if evaluation.DriftScore >= 0.3 {
		evaluation.Status = "drifted"
	} else if evaluation.DriftScore >= 0.15 {
		evaluation.Status = "warning"
	}
	metrics.IntelligenceDriftScore.WithLabelValues(tenantID, label).Set(evaluation.DriftScore)
	metrics.IntelligenceCalibrationError.WithLabelValues(tenantID, label).Set(evaluation.CalibrationError)
	health := 1.0
	if evaluation.Status == "warning" {
		health = 0.5
	} else if evaluation.Status == "drifted" {
		health = 0
	}
	metrics.IntelligenceHealth.WithLabelValues(tenantID, label).Set(health)
	details, _ := json.Marshal(map[string]any{"algorithm": trained.Algorithm, "model_version": trained.Version, "scope": map[bool]string{true: "tenant", false: "global"}[tenantID != ""], "brier_score": evaluation.BrierScore, "calibration_error": evaluation.CalibrationError})
	evaluation.Details = string(details)
	if err := s.DB.Create(&evaluation).Error; err != nil {
		return model.IntelligenceEvaluation{}, err
	}
	return evaluation, nil
}

func (s *UserIntelligenceService) LatestEvaluation(tenantID, label string) (model.IntelligenceEvaluation, error) {
	var evaluation model.IntelligenceEvaluation
	err := s.DB.Where("tenant_id = ? AND label = ?", tenantID, label).Order("created_at desc").First(&evaluation).Error
	return evaluation, err
}

func (s *UserIntelligenceService) Models(tenantID, label string, limit int) ([]model.IntelligenceModel, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	query := s.DB.Where("tenant_id = ?", tenantID).Order("version desc").Limit(limit)
	if label != "" {
		query = query.Where("label = ?", label)
	}
	var models []model.IntelligenceModel
	return models, query.Find(&models).Error
}

func (s *UserIntelligenceService) ModelLayers(tenantID, label string, limit int) ([]model.IntelligenceModel, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := s.DB.Where("tenant_id = ? OR tenant_id = ?", "", tenantID).Order("created_at desc").Limit(limit)
	if label != "" {
		query = query.Where("label = ?", label)
	}
	var models []model.IntelligenceModel
	return models, query.Find(&models).Error
}

func (s *UserIntelligenceService) Evaluations(tenantID, label string, limit int) ([]model.IntelligenceEvaluation, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	query := s.DB.Where("tenant_id = ?", tenantID).Order("created_at desc").Limit(limit)
	if label != "" {
		query = query.Where("label = ?", label)
	}
	var evaluations []model.IntelligenceEvaluation
	return evaluations, query.Find(&evaluations).Error
}

func (s *UserIntelligenceService) PromoteLatest(tenantID, label string, now time.Time) (model.IntelligenceModel, error) {
	var candidate model.IntelligenceModel
	if err := s.DB.Where("tenant_id = ? AND label = ?", tenantID, label).Order("version desc").First(&candidate).Error; err != nil {
		return candidate, err
	}
	var evaluation model.IntelligenceEvaluation
	if err := s.DB.Where("tenant_id = ? AND label = ? AND model_version = ?", tenantID, label, candidate.Version).Order("created_at desc").First(&evaluation).Error; err != nil {
		return candidate, fmt.Errorf("model must be evaluated before promotion: %w", err)
	}
	if evaluation.Status == "drifted" {
		return candidate, fmt.Errorf("model has drift score %.3f and cannot be promoted", evaluation.DriftScore)
	}
	var active model.IntelligenceModel
	if err := s.DB.Where("tenant_id = ? AND label = ? AND status = ?", tenantID, label, "promoted").First(&active).Error; err == nil && active.ID != candidate.ID {
		_ = s.DB.Model(&active).Updates(map[string]any{"status": "rolled_back"})
		candidate.ParentID = active.ID
	}
	candidate.Status = "promoted"
	candidate.PromotedAt = &now
	if err := s.DB.Model(&candidate).Updates(map[string]any{"status": candidate.Status, "promoted_at": candidate.PromotedAt, "parent_id": candidate.ParentID}).Error; err != nil {
		return candidate, err
	}
	if label == "behavior" {
		if err := s.ApplyBehaviorLayers(tenantID); err != nil {
			return candidate, err
		}
	} else if err := s.ApplyModel(candidate); err != nil {
		return candidate, err
	}
	return candidate, nil
}

func (s *UserIntelligenceService) Rollback(tenantID, label string) (model.IntelligenceModel, error) {
	var active model.IntelligenceModel
	if err := s.DB.Where("tenant_id = ? AND label = ? AND status = ?", tenantID, label, "promoted").First(&active).Error; err != nil {
		return active, err
	}
	if active.ParentID == "" {
		return active, fmt.Errorf("no previous promoted model is available")
	}
	var previous model.IntelligenceModel
	if err := s.DB.Where("id = ?", active.ParentID).First(&previous).Error; err != nil {
		return active, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&active).Update("status", "rolled_back").Error; err != nil {
			return err
		}
		return tx.Model(&previous).Updates(map[string]any{"status": "promoted", "promoted_at": time.Now()}).Error
	}); err != nil {
		return active, err
	}
	return previous, nil
}

type LabelSuggestion struct {
	TenantID    string                   `json:"tenant_id"`
	Fingerprint string                   `json:"fingerprint"`
	Label       string                   `json:"label"`
	Confidence  float64                  `json:"confidence"`
	Reasons     []string                 `json:"reasons"`
	Evidence    map[string]float64       `json:"evidence"`
	Identity    string                   `json:"identity"`
	ModelSource string                   `json:"model_source"`
	Snapshot    UserIntelligenceSnapshot `json:"snapshot"`
}

func (s *UserIntelligenceService) LabelSuggestions(tenantID string, limit int) ([]LabelSuggestion, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var rows []model.UserIntelligence
	if err := s.DB.Where("tenant_id = ?", tenantID).Find(&rows).Error; err != nil {
		return nil, err
	}
	var labels []model.UserIntelligenceLabel
	if err := s.DB.Where("tenant_id = ?", tenantID).Find(&labels).Error; err != nil {
		return nil, err
	}
	reviewed := map[string]bool{}
	for _, row := range labels {
		reviewed[row.Fingerprint+":"+row.Label] = true
	}
	result := make([]LabelSuggestion, 0)
	for _, row := range rows {
		var tags []string
		var scores map[string]float64
		_ = json.Unmarshal([]byte(row.Tags), &tags)
		_ = json.Unmarshal([]byte(row.Scores), &scores)
		candidates := []string{"buyer", "potential_buyer", "returning_user", "bot", "automation", "ai_crawler", "high_intent", "inactive"}
		for _, label := range candidates {
			if reviewed[row.Fingerprint+":"+label] {
				continue
			}
			confidence := scores[label]
			if confidence == 0 {
				for _, tag := range tags {
					if tag == label {
						confidence = 0.85
					}
				}
			}
			if confidence < 0.7 {
				continue
			}
			reasons := make([]string, 0, 3)
			for _, tag := range tags {
				if tag == label {
					reasons = append(reasons, "behavioral profile contains "+label)
				}
			}
			if label == "bot" || label == "automation" || label == "ai_crawler" {
				reasons = append(reasons, "user-agent classification indicates automated traffic")
			}
			if label == "buyer" || label == "potential_buyer" || label == "high_intent" {
				reasons = append(reasons, "purchase and intent events were observed")
			}
			if label == "returning_user" {
				reasons = append(reasons, "multiple sessions were observed")
			}
			if label == "inactive" {
				reasons = append(reasons, "last activity is outside the active window")
			}
			evidence := map[string]float64{}
			for key, value := range featuresFromSnapshot(row) {
				evidence[key] = value
			}
			identity := "inferred_fingerprint"
			if row.UserID != "" {
				identity = "explicit_user_id"
			}
			modelSource := "tenant_behavior_layer"
			if scores["global_behavior"] > 0 && scores["tenant_behavior"] == 0 {
				modelSource = "global_behavior_baseline"
			}
			result = append(result, LabelSuggestion{TenantID: tenantID, Fingerprint: row.Fingerprint, Label: label, Confidence: confidence, Reasons: reasons, Evidence: evidence, Identity: identity, ModelSource: modelSource, Snapshot: snapshotOf(row)})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Confidence > result[j].Confidence })
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func featuresFromSnapshot(row model.UserIntelligence) map[string]float64 {
	var features map[string]float64
	_ = json.Unmarshal([]byte(row.Features), &features)
	if features == nil {
		features = map[string]float64{}
	}
	features["anomaly_score"] = row.AnomalyScore
	features["prediction_confidence"] = row.Confidence
	return features
}

func snapshotOf(row model.UserIntelligence) UserIntelligenceSnapshot {
	var tags []string
	var scores, features map[string]float64
	_ = json.Unmarshal([]byte(row.Tags), &tags)
	_ = json.Unmarshal([]byte(row.Scores), &scores)
	_ = json.Unmarshal([]byte(row.Features), &features)
	return UserIntelligenceSnapshot{TenantID: row.TenantID, Fingerprint: row.Fingerprint, UserID: row.UserID, IP: row.IP, UserAgent: row.UserAgent, Tags: tags, Scores: scores, Features: features, AnomalyScore: row.AnomalyScore, PredictedNextEvent: row.PredictedNextEvent, PredictedEventsToday: row.PredictedEventsToday, Confidence: row.Confidence, ComputedAt: row.ComputedAt}
}

// TrainUnsupervised builds a tenant baseline from the explainable snapshots
// already produced by Refresh. It needs no hand-labelled users: the existing
// behavioral tags and anomaly scores become the initial segments, while the
// persisted run gives operators an auditable first-run result.
func (s *UserIntelligenceService) TrainUnsupervised(tenantID string, now time.Time) (model.IntelligenceModel, error) {
	var base model.IntelligenceModel
	if err := s.DB.Where("tenant_id = ? AND label = ?", "", "behavior").Order("version desc").First(&base).Error; err != nil && err != gorm.ErrRecordNotFound {
		return model.IntelligenceModel{}, err
	}
	return s.trainUnsupervisedScope(tenantID, now, base.Version)
}

// TrainUnsupervisedGlobal builds the general behavioral base from every
// tenant. Tenant models inherit its version and only add local adjustments.
func (s *UserIntelligenceService) TrainUnsupervisedGlobal(now time.Time) (model.IntelligenceModel, error) {
	return s.trainUnsupervisedScope("", now, 0)
}

func (s *UserIntelligenceService) trainUnsupervisedScope(tenantID string, now time.Time, baseVersion int) (model.IntelligenceModel, error) {
	var snapshots []model.UserIntelligence
	query := s.DB
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Find(&snapshots).Error; err != nil {
		return model.IntelligenceModel{}, err
	}
	if len(snapshots) == 0 {
		return model.IntelligenceModel{}, fmt.Errorf("unsupervised training requires intelligence snapshots")
	}
	var events []model.Event
	eventQuery := s.DB.Where("is_aggregated = ?", false)
	if tenantID != "" {
		eventQuery = eventQuery.Where("tenant_id = ?", tenantID)
	}
	if err := eventQuery.Find(&events).Error; err != nil {
		return model.IntelligenceModel{}, err
	}
	eventTypes := map[string]int{}
	sessions := map[string]bool{}
	for _, event := range events {
		eventTypes[event.EventName]++
		sessions[event.SessionID] = true
	}
	segments := map[string]int{}
	anomalies := 0
	for _, snapshot := range snapshots {
		var tags []string
		_ = json.Unmarshal([]byte(snapshot.Tags), &tags)
		for _, tag := range tags {
			segments[tag]++
		}
		if snapshot.AnomalyScore >= 0.7 {
			anomalies++
		}
	}
	metrics := map[string]any{
		"profiles":           len(snapshots),
		"events":             len(events),
		"sessions":           len(sessions),
		"event_types":        eventTypes,
		"segments":           segments,
		"anomalies":          anomalies,
		"method":             "behavioral_baseline",
		"base_model_version": baseVersion,
	}
	encoded, _ := json.Marshal(metrics)
	var latest model.IntelligenceModel
	_ = s.DB.Where("tenant_id = ? AND label = ?", tenantID, "behavior").Order("version desc").First(&latest)
	trained := model.IntelligenceModel{ID: fmt.Sprintf("%s:behavior:%d", tenantID, latest.Version+1), TenantID: tenantID, Label: "behavior", Version: latest.Version + 1, Algorithm: "unsupervised_behavior_v1", Parameters: string(encoded), Metrics: string(encoded), Status: "candidate", CreatedAt: now}
	run := model.IntelligenceTrainingRun{ID: fmt.Sprintf("%s:behavior:%d", tenantID, now.UnixNano()), TenantID: tenantID, Label: "behavior", Examples: len(events), Status: "completed", Metrics: string(encoded), CreatedAt: now}
	if err := s.DB.Create(&run).Error; err != nil {
		return model.IntelligenceModel{}, err
	}
	if err := s.DB.Create(&trained).Error; err != nil {
		return model.IntelligenceModel{}, err
	}
	return trained, nil
}

type trainedFeatureStats struct {
	Mean     float64 `json:"mean"`
	Variance float64 `json:"variance"`
}
type trainedModelParameters struct {
	Positive      map[string]trainedFeatureStats `json:"positive"`
	Negative      map[string]trainedFeatureStats `json:"negative"`
	PositivePrior float64                        `json:"positive_prior"`
}

var allowedIntelligenceLabels = map[string]bool{"buyer": true, "potential_buyer": true, "returning_user": true, "new_user": true, "bot": true, "automation": true, "ai_crawler": true, "high_intent": true, "inactive": true}

func (s *UserIntelligenceService) AddLabel(tenantID, fingerprint, label, source string) error {
	if !allowedIntelligenceLabels[label] {
		return fmt.Errorf("unsupported intelligence label: %s", label)
	}
	if source == "" {
		source = "admin"
	}
	id := tenantID + ":" + fingerprint + ":" + label
	return s.DB.Where("id = ?", id).Assign(model.UserIntelligenceLabel{ID: id, TenantID: tenantID, Fingerprint: fingerprint, Label: label, Source: source, Decision: "accepted", CreatedAt: time.Now()}).FirstOrCreate(&model.UserIntelligenceLabel{}).Error
}

func (s *UserIntelligenceService) ReviewSuggestion(tenantID, fingerprint, label, decision string) error {
	if decision != "accepted" && decision != "denied" {
		return fmt.Errorf("decision must be accepted or denied")
	}
	if !allowedIntelligenceLabels[label] {
		return fmt.Errorf("unsupported intelligence label: %s", label)
	}
	id := tenantID + ":" + fingerprint + ":" + label
	row := model.UserIntelligenceLabel{}
	err := s.DB.Where("id = ?", id).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return s.DB.Create(&model.UserIntelligenceLabel{ID: id, TenantID: tenantID, Fingerprint: fingerprint, Label: label, Source: "suggestion", Decision: decision, CreatedAt: time.Now()}).Error
	}
	if err != nil {
		return err
	}
	return s.DB.Model(&row).Updates(map[string]any{"source": "suggestion", "decision": decision, "created_at": time.Now()}).Error
}

func (s *UserIntelligenceService) TrainingReadiness(tenantID, label string) (map[string]any, error) {
	var positive, total int64
	q := s.DB.Model(&model.UserIntelligenceLabel{}).Where("tenant_id = ? AND label = ? AND (decision IS NULL OR decision <> ?)", tenantID, label, "denied")
	if err := q.Count(&positive).Error; err != nil {
		return nil, err
	}
	if err := s.DB.Model(&model.UserIntelligenceLabel{}).Where("tenant_id = ? AND (decision IS NULL OR decision <> ?)", tenantID, "denied").Count(&total).Error; err != nil {
		return nil, err
	}
	return map[string]any{"tenant_id": tenantID, "label": label, "examples": total, "positive": positive, "negative": total - positive, "ready": total >= 20 && positive >= 5 && total-positive >= 5}, nil
}

func (s *UserIntelligenceService) TrainingRuns(tenantID, label string, limit int) ([]model.IntelligenceTrainingRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var runs []model.IntelligenceTrainingRun
	query := s.DB.Where("tenant_id = ?", tenantID).Order("created_at desc").Limit(limit)
	if label != "" {
		query = query.Where("label = ?", label)
	}
	return runs, query.Find(&runs).Error
}

// Train fits a small offline Gaussian Naive Bayes model over the durable
// feature snapshots. It is intentionally explainable and versioned; a future
// heavier model can consume the same labels and feature contract.
func (s *UserIntelligenceService) Train(tenantID, label string, now time.Time) (model.IntelligenceModel, error) {
	var labels []model.UserIntelligenceLabel
	if err := s.DB.Where("tenant_id = ? AND (decision IS NULL OR decision <> ?)", tenantID, "denied").Find(&labels).Error; err != nil {
		return model.IntelligenceModel{}, err
	}
	positive := map[string][]float64{}
	negative := map[string][]float64{}
	for _, labelRow := range labels {
		var snapshot model.UserIntelligence
		if err := s.DB.Where("tenant_id = ? AND fingerprint = ?", tenantID, labelRow.Fingerprint).First(&snapshot).Error; err != nil {
			continue
		}
		var features map[string]float64
		_ = json.Unmarshal([]byte(snapshot.Features), &features)
		target := negative
		if labelRow.Label == label {
			target = positive
		}
		for key, value := range features {
			target[key] = append(target[key], value)
		}
	}
	if len(positive) == 0 || len(negative) == 0 {
		return model.IntelligenceModel{}, fmt.Errorf("training requires positive and negative examples")
	}
	stats := func(values map[string][]float64) map[string]trainedFeatureStats {
		out := map[string]trainedFeatureStats{}
		for key, list := range values {
			var sum float64
			for _, value := range list {
				sum += value
			}
			mean := sum / float64(len(list))
			var variance float64
			for _, value := range list {
				variance += (value - mean) * (value - mean)
			}
			variance /= float64(len(list))
			if variance < 0.0001 {
				variance = 0.0001
			}
			out[key] = trainedFeatureStats{Mean: mean, Variance: variance}
		}
		return out
	}
	positiveCount, negativeCount := 0, 0
	for _, labelRow := range labels {
		if labelRow.Label == label {
			positiveCount++
		} else {
			negativeCount++
		}
	}
	all := positiveCount + negativeCount
	prior := float64(positiveCount) / float64(all)
	if math.IsNaN(prior) || prior <= 0 || prior >= 1 {
		prior = 0.5
	}
	params := trainedModelParameters{Positive: stats(positive), Negative: stats(negative), PositivePrior: prior}
	encoded, _ := json.Marshal(params)
	var latest model.IntelligenceModel
	_ = s.DB.Where("tenant_id = ? AND label = ?", tenantID, label).Order("version desc").First(&latest)
	trained := model.IntelligenceModel{ID: fmt.Sprintf("%s:%s:%d", tenantID, label, latest.Version+1), TenantID: tenantID, Label: label, Version: latest.Version + 1, Algorithm: "gaussian_naive_bayes", Parameters: string(encoded), Metrics: fmt.Sprintf(`{"examples":%d}`, all), Status: "candidate", CreatedAt: now}
	metrics, metricErr := s.evaluateModel(tenantID, label, params)
	if metricErr != nil {
		return model.IntelligenceModel{}, metricErr
	}
	trained.Metrics = metrics
	run := model.IntelligenceTrainingRun{ID: fmt.Sprintf("%s:%s:%d", tenantID, label, now.UnixNano()), TenantID: tenantID, Label: label, Examples: all, Positive: positiveCount, Negative: negativeCount, Status: "completed", Metrics: metrics, CreatedAt: now}
	if err := s.DB.Create(&run).Error; err != nil {
		return model.IntelligenceModel{}, err
	}
	if err := s.DB.Create(&trained).Error; err != nil {
		return model.IntelligenceModel{}, err
	}
	return trained, nil
}

func modelProbability(features map[string]float64, params trainedModelParameters) float64 {
	likelihood := func(stats map[string]trainedFeatureStats) float64 {
		value := 0.0
		for key, feature := range features {
			stat, ok := stats[key]
			if !ok {
				continue
			}
			value += -0.5*math.Log(2*math.Pi*stat.Variance) - ((feature-stat.Mean)*(feature-stat.Mean))/(2*stat.Variance)
		}
		return value
	}
	positive := math.Log(params.PositivePrior) + likelihood(params.Positive)
	negative := math.Log(1-params.PositivePrior) + likelihood(params.Negative)
	odds := math.Exp(math.Min(50, positive-negative))
	return odds / (1 + odds)
}

func (s *UserIntelligenceService) evaluateModel(tenantID, label string, params trainedModelParameters) (string, error) {
	var labels []model.UserIntelligenceLabel
	if err := s.DB.Where("tenant_id = ? AND (decision IS NULL OR decision <> ?)", tenantID, "denied").Find(&labels).Error; err != nil {
		return "", err
	}
	tp, tn, fp, fn := 0, 0, 0, 0
	for _, row := range labels {
		var snapshot model.UserIntelligence
		if err := s.DB.Where("tenant_id = ? AND fingerprint = ?", tenantID, row.Fingerprint).First(&snapshot).Error; err != nil {
			continue
		}
		var features map[string]float64
		_ = json.Unmarshal([]byte(snapshot.Features), &features)
		predicted := modelProbability(features, params) >= 0.5
		actual := row.Label == label
		switch {
		case predicted && actual:
			tp++
		case predicted && !actual:
			fp++
		case !predicted && actual:
			fn++
		default:
			tn++
		}
	}
	total := tp + tn + fp + fn
	accuracy, precision, recall, f1 := 0.0, 0.0, 0.0, 0.0
	if total > 0 {
		accuracy = float64(tp+tn) / float64(total)
	}
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn)
	}
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	return fmt.Sprintf(`{"examples":%d,"true_positive":%d,"true_negative":%d,"false_positive":%d,"false_negative":%d,"accuracy":%.6f,"precision":%.6f,"recall":%.6f,"f1":%.6f}`, total, tp, tn, fp, fn, accuracy, precision, recall, f1), nil
}

// ApplyModel scores every current tenant snapshot using the persisted model.
func (s *UserIntelligenceService) ApplyModel(trained model.IntelligenceModel) error {
	var params trainedModelParameters
	if err := json.Unmarshal([]byte(trained.Parameters), &params); err != nil {
		return err
	}
	var rows []model.UserIntelligence
	if err := s.DB.Where("tenant_id = ?", trained.TenantID).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		var features, scores map[string]float64
		var tags []string
		_ = json.Unmarshal([]byte(row.Features), &features)
		_ = json.Unmarshal([]byte(row.Scores), &scores)
		_ = json.Unmarshal([]byte(row.Tags), &tags)
		if scores == nil {
			scores = map[string]float64{}
		}
		score := modelProbability(features, params)
		scores[trained.Label] = score
		if score >= 0.7 {
			found := false
			for _, tag := range tags {
				if tag == trained.Label {
					found = true
				}
			}
			if !found {
				tags = append(tags, trained.Label)
			}
		}
		encodedScores, _ := json.Marshal(scores)
		encodedTags, _ := json.Marshal(tags)
		if err := s.DB.Model(&row).Updates(map[string]any{"scores": string(encodedScores), "tags": string(encodedTags)}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *UserIntelligenceService) ApplyArtifact(tenantID, label string, parameters []byte) error {
	return s.ApplyModel(model.IntelligenceModel{TenantID: tenantID, Label: label, Parameters: string(parameters)})
}

func (s *UserIntelligenceService) ExportModel(tenantID, label string, registry ModelRegistry, scope string, now time.Time) (ModelManifest, error) {
	var trained model.IntelligenceModel
	if err := s.DB.Where("tenant_id = ? AND label = ?", tenantID, label).Order("version desc").First(&trained).Error; err != nil {
		return ModelManifest{}, err
	}
	manifest := NewManifest("eye-"+label, fmt.Sprintf("%d", trained.Version), label, trained.Algorithm, "eye-user-features-v1", "model.json", "eye-db", []byte(trained.Parameters), now)
	if err := registry.Store(scope, label, manifest, []byte(trained.Parameters)); err != nil {
		return ModelManifest{}, err
	}
	return manifest, nil
}

type UserIntelligenceSnapshot struct {
	TenantID             string             `json:"tenant_id"`
	Fingerprint          string             `json:"fingerprint"`
	UserID               string             `json:"user_id"`
	IP                   string             `json:"ip"`
	UserAgent            string             `json:"user_agent"`
	Tags                 []string           `json:"tags"`
	Scores               map[string]float64 `json:"scores"`
	Features             map[string]float64 `json:"features"`
	AnomalyScore         float64            `json:"anomaly_score"`
	PredictedNextEvent   string             `json:"predicted_next_event"`
	PredictedEventsToday int64              `json:"predicted_events_today"`
	Confidence           float64            `json:"confidence"`
	ComputedAt           time.Time          `json:"computed_at"`
}

func UserFingerprint(ip, userAgent string) string {
	sum := sha256.Sum256([]byte(ip + "\x00" + userAgent))
	return hex.EncodeToString(sum[:])
}

func (s *UserIntelligenceService) Refresh(tenantID string, now time.Time) (int, error) {
	var sessions []model.Session
	if err := s.DB.Where("tenant_id = ?", tenantID).Find(&sessions).Error; err != nil {
		return 0, err
	}
	if len(sessions) == 0 {
		return 0, nil
	}
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.ID)
	}
	var events []model.Event
	if err := s.DB.Where("session_id IN ? AND is_aggregated = ?", ids, false).Find(&events).Error; err != nil {
		return 0, err
	}
	type aggregate struct {
		sessionCount, eventCount int64
		lastSeen                 time.Time
		names                    map[string]int64
		timestamps               []int64
		ip, ua, userID           string
	}
	groups := map[string]*aggregate{}
	sessionKeys := map[string]string{}
	sessionUserIDs := map[string]string{}
	for _, event := range events {
		if event.UserID != "" {
			sessionUserIDs[event.SessionID] = event.UserID
		}
	}
	for _, session := range sessions {
		userID := session.UserID
		if userID == "" {
			userID = sessionUserIDs[session.ID]
		}
		key := session.IP + "\x00" + session.UserAgent
		if userID != "" {
			key = "user:" + userID
		}
		sessionKeys[session.ID] = key
		if groups[key] == nil {
			groups[key] = &aggregate{names: map[string]int64{}, ip: session.IP, ua: session.UserAgent, userID: userID}
		}
		g := groups[key]
		g.sessionCount++
		if session.LastSeen.After(g.lastSeen) {
			g.lastSeen = session.LastSeen
		}
	}
	for _, event := range events {
		if key, ok := sessionKeys[event.SessionID]; ok {
			g := groups[key]
			g.eventCount++
			g.names[event.EventName]++
			g.timestamps = append(g.timestamps, event.ClientTS)
		}
	}
	count := 0
	for key, g := range groups {
		tags := []string{"new_user"}
		scores := map[string]float64{}
		features := map[string]float64{"sessions": float64(g.sessionCount), "events": float64(g.eventCount)}
		if g.sessionCount > 1 {
			tags = append(tags, "returning_user")
		}
		uaLower := strings.ToLower(g.ua)
		if strings.Contains(uaLower, "bot") || strings.Contains(uaLower, "crawler") || strings.Contains(uaLower, "spider") {
			tags = append(tags, "bot", "automation")
			scores["bot"] = 1
		}
		if strings.Contains(uaLower, "gptbot") || strings.Contains(uaLower, "claudebot") || strings.Contains(uaLower, "anthropic") || strings.Contains(uaLower, "perplexitybot") || strings.Contains(uaLower, "google-extended") || strings.Contains(uaLower, "bytespider") {
			tags = append(tags, "ai_crawler")
			scores["ai_crawler"] = 1
		}
		buyer, intent := false, 0.0
		for name, n := range g.names {
			lower := strings.ToLower(name)
			if strings.Contains(lower, "purchase") || strings.Contains(lower, "checkout") || strings.Contains(lower, "payment") || strings.Contains(lower, "order_completed") {
				buyer = true
			}
			if strings.Contains(lower, "product") || strings.Contains(lower, "cart") || strings.Contains(lower, "checkout") {
				intent += float64(n)
			}
		}
		if buyer {
			tags = append(tags, "buyer")
			scores["buyer"] = 1
		} else if intent > 0 {
			tags = append(tags, "potential_buyer")
			scores["buyer"] = math.Min(0.95, intent/10)
		}
		if intent >= 2 {
			tags = append(tags, "high_intent")
		}
		if !g.lastSeen.IsZero() && g.lastSeen.Before(now.Add(-30*24*time.Hour)) {
			tags = append(tags, "inactive")
		}
		anomaly := 0.0
		if g.eventCount >= 1000 && g.sessionCount <= 2 {
			anomaly = math.Min(1, float64(g.eventCount)/10000)
			tags = append(tags, "anomalous")
		}
		if len(g.timestamps) > 0 {
			sort.Slice(g.timestamps, func(i, j int) bool { return g.timestamps[i] < g.timestamps[j] })
		}
		next := ""
		best := int64(0)
		for name, n := range g.names {
			if n > best {
				best, next = n, name
			}
		}
		predicted := int64(math.Round(float64(g.eventCount) * 2))
		confidence := math.Min(1, float64(g.eventCount)/10)
		identityKey := g.ip + "\x00" + g.ua
		if g.userID != "" {
			identityKey = "user:" + g.userID
		}
		row := model.UserIntelligence{ID: tenantID + ":" + UserFingerprint(identityKey, ""), TenantID: tenantID, UserID: g.userID, Fingerprint: UserFingerprint(identityKey, ""), IP: g.ip, UserAgent: g.ua, AnomalyScore: anomaly, PredictedNextEvent: next, PredictedEventsToday: predicted, Confidence: confidence, ComputedAt: now}
		encodedTags, _ := json.Marshal(tags)
		encodedScores, _ := json.Marshal(scores)
		encodedFeatures, _ := json.Marshal(features)
		row.Tags, row.Scores, row.Features = string(encodedTags), string(encodedScores), string(encodedFeatures)
		if err := s.DB.Where("id = ?", row.ID).Assign(row).FirstOrCreate(&row).Error; err != nil {
			return count, err
		}
		count++
		_ = key
	}
	if err := s.ApplyBehaviorLayers(tenantID); err != nil {
		return count, err
	}
	return count, nil
}

// ApplyBehaviorLayers combines the shared global behavioral baseline with the
// latest tenant adaptation. The model stores aggregate segment frequencies,
// so runtime scoring remains explainable and deterministic for every profile.
func (s *UserIntelligenceService) ApplyBehaviorLayers(tenantID string) error {
	var global, local model.IntelligenceModel
	globalQuery := s.DB.Where("tenant_id = ? AND label = ? AND status = ?", "", "behavior", "promoted").Order("version desc")
	if err := globalQuery.First(&global).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	if err := s.DB.Where("tenant_id = ? AND label = ? AND status = ?", tenantID, "behavior", "promoted").Order("version desc").First(&local).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
	}
	globalMetrics := behaviorMetrics(global.Metrics)
	localMetrics := behaviorMetrics(local.Metrics)
	var rows []model.UserIntelligence
	if err := s.DB.Where("tenant_id = ?", tenantID).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		var tags []string
		var scores map[string]float64
		_ = json.Unmarshal([]byte(row.Tags), &tags)
		_ = json.Unmarshal([]byte(row.Scores), &scores)
		if scores == nil {
			scores = map[string]float64{}
		}
		common, rare := false, false
		for _, tag := range tags {
			frequency := globalMetrics.segmentFrequency(tag)
			if frequency >= 0.5 {
				common = true
			}
			if frequency > 0 && frequency < 0.1 {
				rare = true
			}
		}
		if common && !containsTag(tags, "common_behavior") {
			tags = append(tags, "common_behavior")
		}
		if rare && !containsTag(tags, "rare_behavior") {
			tags = append(tags, "rare_behavior")
		}
		globalRate := globalMetrics.anomalyRate()
		localRate := localMetrics.anomalyRate()
		combinedRate := globalRate
		if local.Version > 0 {
			combinedRate = (globalRate + localRate) / 2
			scores["tenant_behavior"] = localRate
		}
		scores["global_behavior"] = globalRate
		row.AnomalyScore = math.Max(row.AnomalyScore, combinedRate)
		row.Confidence = math.Min(1, row.Confidence+0.1)
		encodedTags, _ := json.Marshal(tags)
		encodedScores, _ := json.Marshal(scores)
		row.Tags, row.Scores = string(encodedTags), string(encodedScores)
		if err := s.DB.Save(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

type behaviorModelMetrics struct {
	Profiles  float64
	Anomalies float64
	Segments  map[string]float64
}

func behaviorMetrics(raw string) behaviorModelMetrics {
	var value struct {
		Profiles  float64        `json:"profiles"`
		Anomalies float64        `json:"anomalies"`
		Segments  map[string]int `json:"segments"`
	}
	_ = json.Unmarshal([]byte(raw), &value)
	segments := map[string]float64{}
	for key, count := range value.Segments {
		segments[key] = float64(count)
	}
	return behaviorModelMetrics{Profiles: value.Profiles, Anomalies: value.Anomalies, Segments: segments}
}

func (m behaviorModelMetrics) anomalyRate() float64 {
	if m.Profiles <= 0 {
		return 0
	}
	return math.Min(1, m.Anomalies/m.Profiles)
}

func (m behaviorModelMetrics) segmentFrequency(tag string) float64 {
	if m.Profiles <= 0 {
		return 0
	}
	return m.Segments[tag] / m.Profiles
}

func containsTag(tags []string, wanted string) bool {
	for _, tag := range tags {
		if tag == wanted {
			return true
		}
	}
	return false
}

func (s *UserIntelligenceService) List(tenantID string, limit int) ([]UserIntelligenceSnapshot, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var rows []model.UserIntelligence
	if err := s.DB.Where("tenant_id = ?", tenantID).Order("computed_at desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]UserIntelligenceSnapshot, 0, len(rows))
	for _, row := range rows {
		var snapshot UserIntelligenceSnapshot
		snapshot.TenantID, snapshot.Fingerprint, snapshot.IP, snapshot.UserAgent = row.TenantID, row.Fingerprint, row.IP, row.UserAgent
		snapshot.AnomalyScore, snapshot.PredictedNextEvent, snapshot.PredictedEventsToday, snapshot.Confidence, snapshot.ComputedAt = row.AnomalyScore, row.PredictedNextEvent, row.PredictedEventsToday, row.Confidence, row.ComputedAt
		_ = json.Unmarshal([]byte(row.Tags), &snapshot.Tags)
		_ = json.Unmarshal([]byte(row.Scores), &snapshot.Scores)
		_ = json.Unmarshal([]byte(row.Features), &snapshot.Features)
		out = append(out, snapshot)
	}
	return out, nil
}
