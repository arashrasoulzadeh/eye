package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// AlertService provides CRUD for AlertRule and the threshold-evaluation logic
// that reads it against the Phase 7 materialized AggregateCount read model.
type AlertService struct {
	DB                  *gorm.DB
	DeadLetterNotifyURL string
}

func randomAlertID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *AlertService) Create(tenantID, eventName, dimsKey, metric string, threshold int64, webhookURL string) (*model.AlertRule, error) {
	return a.CreateWithWindow(tenantID, eventName, dimsKey, metric, threshold, 0, webhookURL)
}

func (a *AlertService) CreateWithWindow(tenantID, eventName, dimsKey, metric string, threshold, windowSeconds int64, webhookURL string) (*model.AlertRule, error) {
	return a.CreateWithWindowAndDestination(tenantID, eventName, dimsKey, metric, threshold, windowSeconds, webhookURL, "webhook")
}

func (a *AlertService) CreateWithWindowAndDestination(tenantID, eventName, dimsKey, metric string, threshold, windowSeconds int64, webhookURL, destination string) (*model.AlertRule, error) {
	if metric == "" {
		metric = "count"
	}
	if err := validateAlertRule(eventName, metric, threshold, windowSeconds); err != nil {
		return nil, err
	}
	if err := validateWebhookURL(webhookURL); err != nil {
		return nil, err
	}
	destination, err := normalizeAlertDestination(destination)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	rule := &model.AlertRule{
		ID: randomAlertID(), TenantID: tenantID, EventName: eventName, DimsKey: dimsKey,
		Metric: metric, Threshold: threshold, WindowSeconds: windowSeconds, WebhookURL: webhookURL, Destination: destination, Enabled: true,
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
	return a.UpdateWithWindow(id, eventName, dimsKey, metric, threshold, 0, webhookURL)
}

func (a *AlertService) UpdateWithWindow(id, eventName, dimsKey, metric string, threshold, windowSeconds int64, webhookURL string) (*model.AlertRule, error) {
	return a.UpdateWithWindowAndDestination(id, eventName, dimsKey, metric, threshold, windowSeconds, webhookURL, "")
}

func (a *AlertService) UpdateWithWindowAndDestination(id, eventName, dimsKey, metric string, threshold, windowSeconds int64, webhookURL, destination string) (*model.AlertRule, error) {
	if metric == "" {
		metric = "count"
	}
	if err := validateAlertRule(eventName, metric, threshold, windowSeconds); err != nil {
		return nil, err
	}
	if err := validateWebhookURL(webhookURL); err != nil {
		return nil, err
	}
	if destination == "" {
		destination = "webhook"
	}
	destination, err := normalizeAlertDestination(destination)
	if err != nil {
		return nil, err
	}
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
	if windowSeconds > 0 || metric == "absence" {
		rule.WindowSeconds = windowSeconds
	}
	rule.WebhookURL = webhookURL
	rule.Destination = destination
	rule.UpdatedAt = time.Now().Unix()
	if err := a.DB.Save(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

func normalizeAlertDestination(destination string) (string, error) {
	destination = strings.ToLower(strings.TrimSpace(destination))
	if destination == "" {
		destination = "webhook"
	}
	switch destination {
	case "webhook", "slack", "pagerduty", "email":
		return destination, nil
	default:
		return "", fmt.Errorf("unsupported alert destination %q", destination)
	}
}

func validateAlertRule(eventName, metric string, threshold, windowSeconds int64) error {
	if strings.TrimSpace(eventName) == "" {
		return fmt.Errorf("event name is required")
	}
	if metric != "count" && metric != "absence" && metric != "conversion" {
		return fmt.Errorf("unsupported alert metric %q", metric)
	}
	if metric != "absence" && threshold <= 0 {
		return fmt.Errorf("threshold must be greater than zero")
	}
	if (metric == "absence" || metric == "conversion") && windowSeconds < 0 {
		return fmt.Errorf("window_seconds cannot be negative")
	}
	if metric == "conversion" && len(strings.Split(eventName, ",")) < 2 {
		return fmt.Errorf("conversion requires at least two comma-separated events")
	}
	return nil
}

func validateWebhookURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("webhook_url must be an absolute http(s) URL")
	}
	return nil
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
	if rule.Metric == "conversion" {
		names := strings.Split(rule.EventName, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		if len(names) < 2 {
			return false, 0, nil
		}
		window := rule.WindowSeconds
		if window <= 0 {
			window = 86400
		}
		cutoff := time.Now().Add(-time.Duration(window) * time.Second)
		var events []model.Event
		if err := a.DB.Where("tenant_id = ? AND ingest_ts >= ?", rule.TenantID, cutoff).Where("event_name IN ?", names).Find(&events).Error; err != nil {
			return false, 0, err
		}
		completed := make(map[string]map[string]bool)
		for _, event := range events {
			if completed[event.SessionID] == nil {
				completed[event.SessionID] = map[string]bool{}
			}
			completed[event.SessionID][event.EventName] = true
		}
		for _, seen := range completed {
			ok := true
			for _, name := range names {
				if !seen[name] {
					ok = false
					break
				}
			}
			if ok {
				count++
			}
		}
		return count >= rule.Threshold, count, nil
	}
	if rule.Metric == "absence" {
		window := rule.WindowSeconds
		if window <= 0 {
			window = 300
		}
		cutoff := time.Now().Add(-time.Duration(window) * time.Second)
		var total int64
		q := a.DB.Model(&model.Event{}).Where("tenant_id = ? AND event_name = ? AND ingest_ts >= ?", rule.TenantID, rule.EventName, cutoff)
		if rule.DimsKey != "" {
			q = q.Where("dims = ?", rule.DimsKey)
		}
		if err := q.Count(&total).Error; err != nil {
			return false, 0, err
		}
		return total == 0, total, nil
	}
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
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned HTTP %s", resp.Status)
	}
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
			"rule_id":        rule.ID,
			"tenant_id":      rule.TenantID,
			"event_name":     rule.EventName,
			"dims_key":       rule.DimsKey,
			"metric":         rule.Metric,
			"threshold":      rule.Threshold,
			"count":          count,
			"window_seconds": rule.WindowSeconds,
			"fired_at":       time.Now().Unix(),
		}
		// Phase 14: retries + dead-letter queue. A transient failure gets a
		// few attempts with a short backoff before giving up; if every
		// attempt fails, DispatchWithRetry persists the delivery to
		// WebhookDeadLetter instead of this just silently dropping it.
		deliveryPayload := alertDeliveryPayload(rule.Destination, payload)
		if err := DispatchWithRetryAndNotify(svc.DB, client, svc.DeadLetterNotifyURL, rule.ID, rule.WebhookURL, deliveryPayload, 3, func(attempt int) {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}); err != nil {
			if logger != nil {
				logger.Error("alert job: webhook dispatch failed after retries, dead-lettered", "id", rule.ID, "err", err)
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

func alertDeliveryPayload(destination string, payload map[string]any) map[string]any {
	switch strings.ToLower(destination) {
	case "slack":
		b, _ := json.Marshal(payload)
		return map[string]any{"text": string(b), "eye": payload}
	case "pagerduty":
		return map[string]any{"event_action": "trigger", "payload": map[string]any{
			"summary": "Eye alert fired", "source": "eye", "severity": "error", "custom_details": payload,
		}}
	default:
		return payload
	}
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
