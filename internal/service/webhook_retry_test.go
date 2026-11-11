package service

import (
	"context"
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
