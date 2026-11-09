package test

import (
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func newWebhookRetryTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.AlertRule{}, &model.AggregateCount{}, &model.WebhookDeadLetter{}); err != nil {
		t.Fatal(err)
	}
	return db
}

// alwaysFailClient fails every PostJSON call.
type alwaysFailClient struct{ calls int }

func (c *alwaysFailClient) PostJSON(url string, payload map[string]any) error {
	c.calls++
	return errors.New("connection refused")
}

// failThenSucceedClient fails failures times, then succeeds.
type failThenSucceedClient struct {
	failures int
	calls    int
}

func (c *failThenSucceedClient) PostJSON(url string, payload map[string]any) error {
	c.calls++
	if c.calls <= c.failures {
		return errors.New("timeout")
	}
	return nil
}

func TestDispatchWithRetryExhaustsAndDeadLetters(t *testing.T) {
	db := newWebhookRetryTestDB(t)
	client := &alwaysFailClient{}
	maxAttempts := 3

	err := service.DispatchWithRetry(db, client, "rule-1", "https://example.com/hook",
		map[string]any{"foo": "bar"}, maxAttempts, func(attempt int) {})

	if err == nil {
		t.Fatal("expected an error after exhausting retries")
	}
	if client.calls != maxAttempts {
		t.Fatalf("expected %d attempts, got %d", maxAttempts, client.calls)
	}

	var dead []model.WebhookDeadLetter
	db.Find(&dead)
	if len(dead) != 1 {
		t.Fatalf("expected 1 dead-lettered delivery, got %d", len(dead))
	}
	if dead[0].Attempts != maxAttempts {
		t.Fatalf("expected recorded attempts %d, got %d", maxAttempts, dead[0].Attempts)
	}
	if dead[0].AlertRuleID != "rule-1" {
		t.Fatalf("expected rule-1, got %s", dead[0].AlertRuleID)
	}
}

func TestDispatchWithRetrySucceedsWithoutDeadLetter(t *testing.T) {
	db := newWebhookRetryTestDB(t)
	client := &failThenSucceedClient{failures: 2}

	err := service.DispatchWithRetry(db, client, "rule-2", "https://example.com/hook",
		map[string]any{}, 5, func(attempt int) {})

	if err != nil {
		t.Fatalf("expected eventual success, got %v", err)
	}
	if client.calls != 3 {
		t.Fatalf("expected 3 calls (2 fail + 1 success), got %d", client.calls)
	}
	var count int64
	db.Model(&model.WebhookDeadLetter{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected no dead-lettered rows on eventual success, got %d", count)
	}
}

func TestDispatchWithRetryBackoffCallback(t *testing.T) {
	db := newWebhookRetryTestDB(t)
	client := &alwaysFailClient{}
	var sleptAttempts []int

	service.DispatchWithRetry(db, client, "rule-3", "https://example.com/hook",
		map[string]any{}, 3, func(attempt int) { sleptAttempts = append(sleptAttempts, attempt) })

	// Backoff is invoked between attempts, so 3 attempts means 2 backoff calls.
	if len(sleptAttempts) != 2 {
		t.Fatalf("expected 2 backoff invocations between 3 attempts, got %d: %v", len(sleptAttempts), sleptAttempts)
	}
}

func TestWebhookDeadLetterHandlerLists(t *testing.T) {
	db := newWebhookRetryTestDB(t)
	client := &alwaysFailClient{}
	service.DispatchWithRetry(db, client, "rule-4", "https://example.com/hook", map[string]any{"x": 1}, 2, func(int) {})

	svc := &service.WebhookDeadLetterService{DB: db}
	list, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 dead-lettered entry, got %d", len(list))
	}
	if list[0].LastError == "" {
		t.Fatal("expected LastError to be recorded")
	}
}
