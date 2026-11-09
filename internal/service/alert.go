package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

// AlertService provides CRUD for AlertRule and the threshold-evaluation logic
// that reads it against the Phase 7 materialized AggregateCount read model.
type AlertService struct{ DB *gorm.DB }

func randomAlertID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *AlertService) Create(tenantID, eventName, dimsKey, metric string, threshold int64, webhookURL string) (*model.AlertRule, error) {
	if metric == "" {
		metric = "count"
	}
	now := time.Now().Unix()
	rule := &model.AlertRule{
		ID: randomAlertID(), TenantID: tenantID, EventName: eventName, DimsKey: dimsKey,
		Metric: metric, Threshold: threshold, WebhookURL: webhookURL, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := a.DB.Create(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

func (a *AlertService) List(tenantID string) ([]model.AlertRule, error) {
	var list []model.AlertRule
	if err := a.DB.Where("tenant_id = ?", tenantID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (a *AlertService) Get(id string) (*model.AlertRule, error) {
	var rule model.AlertRule
	if err := a.DB.First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (a *AlertService) Update(id, eventName, dimsKey, metric string, threshold int64, webhookURL string) (*model.AlertRule, error) {
	rule, err := a.Get(id)
	if err != nil {
		return nil, err
	}
	rule.EventName = eventName
	rule.DimsKey = dimsKey
	if metric != "" {
		rule.Metric = metric
	}
	rule.Threshold = threshold
	rule.WebhookURL = webhookURL
	rule.UpdatedAt = time.Now().Unix()
	if err := a.DB.Save(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

func (a *AlertService) Delete(id string) error {
	return a.DB.Delete(&model.AlertRule{}, "id = ?", id).Error
}

// SetEnabled toggles whether a rule is evaluated by RunAlertJob.
func (a *AlertService) SetEnabled(id string, enabled bool) (*model.AlertRule, error) {
	rule, err := a.Get(id)
	if err != nil {
		return nil, err
	}
	rule.Enabled = enabled
	rule.UpdatedAt = time.Now().Unix()
	if err := a.DB.Save(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

// Evaluate checks whether rule's threshold is currently met against the
// materialized AggregateCount totals. Only "count" is a supported metric for
// now. If rule.DimsKey is empty, counts are summed across every distinct
// dims bucket for (TenantID, EventName); otherwise only the matching dims
// bucket is used.
func (a *AlertService) Evaluate(rule *model.AlertRule) (met bool, count int64, err error) {
	q := a.DB.Model(&model.AggregateCount{}).Where("tenant_id = ? AND event_name = ?", rule.TenantID, rule.EventName)
	if rule.DimsKey != "" {
		q = q.Where("dims_key = ?", rule.DimsKey)
	}
	var total int64
	if err := q.Select("COALESCE(SUM(count), 0)").Row().Scan(&total); err != nil {
		return false, 0, err
	}
	return total >= rule.Threshold, total, nil
}

// WebhookClient is the injectable dispatch seam so tests can assert on
// payload/URL without a real network call.
type WebhookClient interface {
	PostJSON(url string, payload map[string]any) error
}

// WebhookCall records one PostJSON invocation; used by fakes in tests.
type WebhookCall struct {
	URL     string
	Payload map[string]any
}

// HTTPWebhookClient is the real implementation, used outside tests.
type HTTPWebhookClient struct{ Client *http.Client }

func (h *HTTPWebhookClient) PostJSON(url string, payload map[string]any) error {
	client := h.Client
	if client == nil {
		client = http.DefaultClient
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// EvaluateAndFire checks every enabled AlertRule across all tenants and
// fires client.PostJSON for each one whose threshold is currently met,
// provided it has not already fired for the current level (LastFiredAt is
// only reset by evaluating again after the count changes below threshold and
// back up — see note below). It returns how many webhooks were actually
// dispatched this call.
//
// This is a polling-based check (see RunAlertJob), not a real-time push: the
// rule only re-fires once its underlying count has grown since the last fire
// (LastFiredAtCount), so a rule sitting above threshold with no new events
// does not spam the webhook on every tick.
func EvaluateAndFire(svc *AlertService, client WebhookClient, logger *slog.Logger) (int, error) {
	var rules []model.AlertRule
	if err := svc.DB.Where("enabled = ?", true).Find(&rules).Error; err != nil {
		return 0, err
	}
	fired := 0
	for i := range rules {
		rule := rules[i]
		met, count, err := svc.Evaluate(&rule)
		if err != nil {
			if logger != nil {
				logger.Error("alert job: evaluate failed", "id", rule.ID, "err", err)
			}
			continue
		}
		if !met {
			continue
		}
		if rule.LastFiredAt != 0 && rule.LastFiredCount >= count {
			// Already fired for this count level; no new events since.
			continue
		}
		payload := map[string]any{
			"rule_id":    rule.ID,
			"tenant_id":  rule.TenantID,
			"event_name": rule.EventName,
			"dims_key":   rule.DimsKey,
			"metric":     rule.Metric,
			"threshold":  rule.Threshold,
			"count":      count,
			"fired_at":   time.Now().Unix(),
		}
		if err := client.PostJSON(rule.WebhookURL, payload); err != nil {
			if logger != nil {
				logger.Error("alert job: webhook dispatch failed", "id", rule.ID, "err", err)
			}
			continue
		}
		rule.LastFiredAt = time.Now().Unix()
		rule.LastFiredCount = count
		if err := svc.DB.Save(&rule).Error; err != nil && logger != nil {
			logger.Error("alert job: failed to persist LastFiredAt", "id", rule.ID, "err", err)
		}
		fired++
	}
	return fired, nil
}

// RunAlertJob periodically evaluates every enabled AlertRule and dispatches
// webhooks for newly-met thresholds. Like RunCohortMaterializationJob and
// RunAggregationJob, this is a simple polling check against the materialized
// AggregateCount table, not a real-time event-driven alert.
func RunAlertJob(ctx context.Context, svc *AlertService, client WebhookClient, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := EvaluateAndFire(svc, client, logger); err != nil && logger != nil {
				logger.Error("alert job: evaluate and fire failed", "err", err)
			}
		}
	}
}
