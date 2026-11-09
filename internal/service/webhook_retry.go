package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

func randomDeadLetterID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// DispatchWithRetry calls client.PostJSON(url, payload), retrying up to
// maxAttempts times on error. backoff is called once between each pair of
// attempts (so maxAttempts attempts means maxAttempts-1 backoff calls) — it
// takes the attempt number just completed, letting callers sleep a real
// delay in production while tests pass a no-op or recording function to
// stay instant and deterministic. If every attempt fails, the delivery is
// persisted to WebhookDeadLetter instead of being silently dropped, and the
// final error is returned so the caller can log it.
func DispatchWithRetry(db *gorm.DB, client WebhookClient, alertRuleID, url string, payload map[string]any, maxAttempts int, backoff func(attempt int)) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = client.PostJSON(url, payload)
		if lastErr == nil {
			return nil
		}
		if attempt < maxAttempts {
			backoff(attempt)
		}
	}

	body, _ := json.Marshal(payload)
	dl := &model.WebhookDeadLetter{
		ID:          randomDeadLetterID(),
		AlertRuleID: alertRuleID,
		URL:         url,
		Payload:     string(body),
		LastError:   lastErr.Error(),
		Attempts:    maxAttempts,
		FailedAt:    time.Now().Unix(),
	}
	_ = db.Create(dl).Error
	return lastErr
}

// WebhookDeadLetterService lists exhausted webhook deliveries so an operator
// can inspect and, if useful, manually re-fire them.
type WebhookDeadLetterService struct{ DB *gorm.DB }

func (s *WebhookDeadLetterService) List() ([]model.WebhookDeadLetter, error) {
	var list []model.WebhookDeadLetter
	if err := s.DB.Order("failed_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
