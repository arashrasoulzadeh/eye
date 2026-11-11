package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func newTestDB_webhook_test(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.WebhookDeadLetter{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

type fakeWebhookClient struct {
	fn func(url string, payload map[string]any) error
}

type recordingWebhookClient struct {
	calls []WebhookCall
	err   error
}

func (r *recordingWebhookClient) PostJSON(url string, payload map[string]any) error {
	r.calls = append(r.calls, WebhookCall{URL: url, Payload: payload})
	return r.err
}

func (f *fakeWebhookClient) PostJSON(url string, payload map[string]any) error {
	return f.fn(url, payload)
}

func TestWebhookDeadLetterService_List(t *testing.T) {
	db := newTestDB_webhook_test(t)
	svc := &WebhookDeadLetterService{DB: db}

	// Create a dead letter
	dl := &model.WebhookDeadLetter{
		ID:          "dl1",
		AlertRuleID: "rule1",
		URL:         "http://webhook",
		Payload:     `{"test": "data"}`,
		LastError:   "timeout",
		Attempts:    3,
		FailedAt:    time.Now().UnixMilli(),
	}
	db.Create(dl)

	list, err := svc.List(100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("list len = %d, want 1", len(list))
	}
	if list[0].ID != "dl1" {
		t.Errorf("list[0].ID = %s, want dl1", list[0].ID)
	}
}

func TestWebhookDeadLetterService_Delete(t *testing.T) {
	db := newTestDB_webhook_test(t)
	svc := &WebhookDeadLetterService{DB: db}
	db.Create(&model.WebhookDeadLetter{ID: "delete-me", URL: "https://example.com"})
	if err := svc.Delete("delete-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&model.WebhookDeadLetter{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected dead letter to be deleted, got %d", count)
	}
}

func TestWebhookDeadLetterService_Replay(t *testing.T) {
	db := newTestDB_webhook_test(t)
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		received = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	svc := &WebhookDeadLetterService{DB: db}
	db.Create(&model.WebhookDeadLetter{ID: "replay-me", URL: server.URL, Payload: `{"event":"retry"}`})
	if err := svc.Replay("replay-me"); err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if received != `{"event":"retry"}` {
		t.Fatalf("unexpected replay payload: %s", received)
	}
	var count int64
	db.Model(&model.WebhookDeadLetter{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected replayed record removed, got %d", count)
	}
}

func TestDispatchWithRetry(t *testing.T) {
	db := newTestDB_webhook_test(t)

	// Test success on first try
	var callCount int
	client := &fakeWebhookClient{fn: func(url string, payload map[string]any) error {
		callCount++
		return nil
	}}

	err := DispatchWithRetry(db, client, "rule1", "http://webhook", map[string]any{"test": "data"}, 3, func(attempt int) {
		time.Sleep(time.Duration(attempt) * 10 * time.Millisecond)
	})
	if err != nil {
		t.Fatalf("DispatchWithRetry: %v", err)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Test failure then success
	callCount = 0
	attempts := 0
	client2 := &fakeWebhookClient{fn: func(url string, payload map[string]any) error {
		attempts++
		if attempts < 2 {
			return context.DeadlineExceeded
		}
		return nil
	}}

	_ = DispatchWithRetry(db, client2, "rule2", "http://webhook", map[string]any{"test": "data"}, 3, func(attempt int) {
		time.Sleep(time.Duration(attempt) * 10 * time.Millisecond)
	})
	if err != nil {
		t.Fatalf("DispatchWithRetry: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestDispatchWithRetryAndNotifyDeadLetter(t *testing.T) {
	db := newTestDB_webhook_test(t)
	client := &recordingWebhookClient{err: errors.New("unavailable")}
	err := DispatchWithRetryAndNotify(db, client, "https://ops.example/dead-letters", "rule-notify", "https://target.example/hook", map[string]any{"event": "x"}, 2, func(int) {})
	if err == nil {
		t.Fatal("expected delivery error")
	}
	if len(client.calls) != 3 {
		t.Fatalf("expected 2 delivery attempts plus notification, got %d", len(client.calls))
	}
	notification := client.calls[2]
	if notification.URL != "https://ops.example/dead-letters" {
		t.Fatalf("notification URL = %q", notification.URL)
	}
	if notification.Payload["type"] != "webhook_dead_letter" || notification.Payload["alert_rule_id"] != "rule-notify" {
		t.Fatalf("unexpected notification payload: %#v", notification.Payload)
	}
}
