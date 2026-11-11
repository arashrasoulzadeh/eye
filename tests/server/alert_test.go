package test

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func newAlertTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	// Pin to a single connection — see cohort_test.go's newCohortTestDB for why
	// a plain ":memory:" DSN otherwise hands out a fresh empty DB per connection.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.AlertRule{}, &model.AggregateCount{}, &model.Event{}, &service.MetricSample{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAlertServiceAnomalyRule(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	for i, count := range []int64{10, 10, 10, 100} {
		db.Create(&service.MetricSample{TenantID: "t1", EventName: "checkout", Bucket: int64(i + 1), Count: count, Cumulative: count})
	}
	rule, err := svc.CreateWithWindow("t1", "checkout", "", "anomaly", 2, 0, "https://example.com/hook")
	if err != nil {
		t.Fatal(err)
	}
	met, count, err := svc.Evaluate(rule)
	if err != nil || !met || count != 100 {
		t.Fatalf("expected anomaly, met=%v count=%d err=%v", met, count, err)
	}
}

func TestAlertServiceAbsenceRule(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	rule, err := svc.CreateWithWindow("t1", "heartbeat", "", "absence", 0, 60, "https://example.com/hook")
	if err != nil {
		t.Fatal(err)
	}
	met, count, err := svc.Evaluate(rule)
	if err != nil {
		t.Fatal(err)
	}
	if !met || count != 0 {
		t.Fatalf("expected absence to be met with zero events, met=%v count=%d", met, count)
	}
	db.Create(&model.Event{EventID: "e1", TenantID: "t1", EventName: "heartbeat", IngestTS: time.Now()})
	met, count, err = svc.Evaluate(rule)
	if err != nil {
		t.Fatal(err)
	}
	if met || count != 1 {
		t.Fatalf("expected recent event to clear absence, met=%v count=%d", met, count)
	}
	updated, err := svc.UpdateWithWindow(rule.ID, "heartbeat", "", "absence", 0, 120, rule.WebhookURL)
	if err != nil || updated.WindowSeconds != 120 {
		t.Fatalf("expected absence window update, rule=%+v err=%v", updated, err)
	}
}

func TestAlertServiceConversionRule(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	now := time.Now()
	db.Create(&model.Event{EventID: "c1", TenantID: "t1", SessionID: "s1", EventName: "landing", IngestTS: now})
	db.Create(&model.Event{EventID: "c2", TenantID: "t1", SessionID: "s1", EventName: "purchase", IngestTS: now})
	db.Create(&model.Event{EventID: "c3", TenantID: "t1", SessionID: "s2", EventName: "landing", IngestTS: now})
	rule, err := svc.CreateWithWindow("t1", "landing,purchase", "", "conversion", 1, 3600, "https://example.com/hook")
	if err != nil {
		t.Fatal(err)
	}
	met, count, err := svc.Evaluate(rule)
	if err != nil {
		t.Fatal(err)
	}
	if !met || count != 1 {
		t.Fatalf("expected one converted session, met=%v count=%d", met, count)
	}
}

func TestAlertServiceRejectsInvalidRules(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	if _, err := svc.CreateWithWindow("t1", "landing", "", "conversion", 1, 60, "https://example.com"); err == nil {
		t.Fatal("expected conversion sequence validation error")
	}
	if _, err := svc.CreateWithWindow("t1", "error", "", "count", 0, 0, "https://example.com"); err == nil {
		t.Fatal("expected threshold validation error")
	}
	if _, err := svc.CreateWithWindow("t1", "error", "", "unknown", 1, 0, "https://example.com"); err == nil {
		t.Fatal("expected metric validation error")
	}
	if _, err := svc.CreateWithWindow("t1", "error", "", "count", 1, 0, "file:///tmp/hook"); err == nil {
		t.Fatal("expected webhook URL validation error")
	}
}

func TestAlertServiceCRUD(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}

	rule, err := svc.Create("t1", "purchase", "", "count", 5, "https://example.com/hook")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if rule.ID == "" {
		t.Fatal("expected a generated ID")
	}
	if !rule.Enabled {
		t.Error("new rule should default to enabled")
	}

	if _, err := svc.Create("t2", "other", "", "count", 1, "https://example.com/hook2"); err != nil {
		t.Fatalf("create for t2: %v", err)
	}

	list, err := svc.List("t1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != rule.ID {
		t.Fatalf("list leaked across tenants or missed rule: %+v", list)
	}

	got, err := svc.Get(rule.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Threshold != 5 {
		t.Fatalf("get returned wrong rule: %+v", got)
	}

	updated, err := svc.Update(rule.ID, "purchase", "", "count", 10, "https://example.com/hook3")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Threshold != 10 || updated.WebhookURL != "https://example.com/hook3" {
		t.Fatalf("update did not persist changes: %+v", updated)
	}

	if err := svc.Delete(rule.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(rule.ID); err == nil {
		t.Fatal("expected Get to fail after Delete")
	}
}

func TestAlertServiceSetEnabled(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	rule, _ := svc.Create("t1", "purchase", "", "count", 5, "https://example.com/hook")

	disabled, err := svc.SetEnabled(rule.ID, false)
	if err != nil {
		t.Fatalf("set enabled false: %v", err)
	}
	if disabled.Enabled {
		t.Fatal("expected rule to be disabled")
	}

	enabled, err := svc.SetEnabled(rule.ID, true)
	if err != nil {
		t.Fatalf("set enabled true: %v", err)
	}
	if !enabled.Enabled {
		t.Fatal("expected rule to be enabled")
	}
}

func TestAlertServiceGetMissing(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	if _, err := svc.Get("does-not-exist"); err == nil {
		t.Fatal("expected error for missing rule")
	}
}

// --- threshold evaluation ---

func TestAlertServiceEvaluate_BelowThreshold(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	rule, _ := svc.Create("t1", "purchase", "", "count", 5, "https://example.com/hook")

	if err := service.UpsertAggregateCount(db, "t1", "purchase", ""); err != nil {
		t.Fatal(err)
	}

	met, count, err := svc.Evaluate(rule)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if met {
		t.Fatalf("expected threshold not met, got count=%d", count)
	}
}

func TestAlertServiceEvaluate_MeetsThreshold(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	rule, _ := svc.Create("t1", "purchase", "", "count", 3, "https://example.com/hook")

	for i := 0; i < 3; i++ {
		if err := service.UpsertAggregateCount(db, "t1", "purchase", ""); err != nil {
			t.Fatal(err)
		}
	}

	met, count, err := svc.Evaluate(rule)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !met {
		t.Fatalf("expected threshold met, got count=%d", count)
	}
	if count != 3 {
		t.Fatalf("expected count=3, got %d", count)
	}
}

func TestAlertServiceEvaluate_SumsAcrossDims(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	// DimsKey "" on the rule means "any dims" -> sum across distinct dims keys.
	rule, _ := svc.Create("t1", "purchase", "", "count", 4, "https://example.com/hook")

	if err := service.UpsertAggregateCount(db, "t1", "purchase", `{"platform":"web"}`); err != nil {
		t.Fatal(err)
	}
	if err := service.UpsertAggregateCount(db, "t1", "purchase", `{"platform":"web"}`); err != nil {
		t.Fatal(err)
	}
	if err := service.UpsertAggregateCount(db, "t1", "purchase", `{"platform":"ios"}`); err != nil {
		t.Fatal(err)
	}
	if err := service.UpsertAggregateCount(db, "t1", "purchase", `{"platform":"ios"}`); err != nil {
		t.Fatal(err)
	}

	met, count, err := svc.Evaluate(rule)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !met || count != 4 {
		t.Fatalf("expected met=true count=4, got met=%v count=%d", met, count)
	}
}

func TestAlertServiceEvaluate_SpecificDims(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	rule, _ := svc.Create("t1", "purchase", `{"platform":"web"}`, "count", 2, "https://example.com/hook")

	if err := service.UpsertAggregateCount(db, "t1", "purchase", `{"platform":"web"}`); err != nil {
		t.Fatal(err)
	}
	if err := service.UpsertAggregateCount(db, "t1", "purchase", `{"platform":"ios"}`); err != nil {
		t.Fatal(err)
	}
	if err := service.UpsertAggregateCount(db, "t1", "purchase", `{"platform":"ios"}`); err != nil {
		t.Fatal(err)
	}

	// Only the "web" dims bucket counts toward this rule, and it only has 1.
	met, count, err := svc.Evaluate(rule)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if met {
		t.Fatalf("expected not met (web bucket has 1, threshold 2), got count=%d", count)
	}
	if count != 1 {
		t.Fatalf("expected count=1 for the web-only bucket, got %d", count)
	}
}

// --- webhook dispatch ---

type fakeHTTPClient struct {
	calls []service.WebhookCall
	err   error
}

func (f *fakeHTTPClient) PostJSON(url string, payload map[string]any) error {
	f.calls = append(f.calls, service.WebhookCall{URL: url, Payload: payload})
	return f.err
}

func TestRunAlertJob_FiresWebhookWhenThresholdMet(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	rule, _ := svc.Create("t1", "purchase", "", "count", 2, "https://example.com/hook")
	for i := 0; i < 2; i++ {
		if err := service.UpsertAggregateCount(db, "t1", "purchase", ""); err != nil {
			t.Fatal(err)
		}
	}

	client := &fakeHTTPClient{}
	fired, err := service.EvaluateAndFire(svc, client, nil)
	if err != nil {
		t.Fatalf("evaluate and fire: %v", err)
	}
	if fired != 1 {
		t.Fatalf("expected 1 webhook fired, got %d", fired)
	}
	if len(client.calls) != 1 {
		t.Fatalf("expected 1 webhook call, got %d", len(client.calls))
	}
	if client.calls[0].URL != "https://example.com/hook" {
		t.Fatalf("wrong webhook URL: %s", client.calls[0].URL)
	}
	if client.calls[0].Payload["rule_id"] != rule.ID {
		t.Fatalf("payload missing rule_id: %+v", client.calls[0].Payload)
	}
	if client.calls[0].Payload["count"] != int64(2) {
		t.Fatalf("payload missing correct count: %+v", client.calls[0].Payload)
	}

	// LastFiredAt should now be set so a second tick without new events doesn't re-fire.
	got, _ := svc.Get(rule.ID)
	if got.LastFiredAt == 0 {
		t.Fatal("expected LastFiredAt to be set after firing")
	}

	fired2, err := service.EvaluateAndFire(svc, client, nil)
	if err != nil {
		t.Fatalf("second evaluate and fire: %v", err)
	}
	if fired2 != 0 {
		t.Fatalf("expected no re-fire while still above threshold with no new events, got %d", fired2)
	}
	if len(client.calls) != 1 {
		t.Fatalf("expected still only 1 webhook call total, got %d", len(client.calls))
	}
}

func TestRunAlertJob_DoesNotFireBelowThreshold(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	svc.Create("t1", "purchase", "", "count", 5, "https://example.com/hook")
	service.UpsertAggregateCount(db, "t1", "purchase", "")

	client := &fakeHTTPClient{}
	fired, err := service.EvaluateAndFire(svc, client, nil)
	if err != nil {
		t.Fatalf("evaluate and fire: %v", err)
	}
	if fired != 0 {
		t.Fatalf("expected 0 fired, got %d", fired)
	}
	if len(client.calls) != 0 {
		t.Fatalf("expected no webhook calls, got %d", len(client.calls))
	}
}

func TestRunAlertJob_SkipsDisabledRules(t *testing.T) {
	db := newAlertTestDB(t)
	svc := &service.AlertService{DB: db}
	rule, _ := svc.Create("t1", "purchase", "", "count", 1, "https://example.com/hook")
	service.UpsertAggregateCount(db, "t1", "purchase", "")

	disabled, err := svc.Get(rule.ID)
	if err != nil {
		t.Fatal(err)
	}
	disabled.Enabled = false
	if err := db.Save(disabled).Error; err != nil {
		t.Fatal(err)
	}

	client := &fakeHTTPClient{}
	fired, err := service.EvaluateAndFire(svc, client, nil)
	if err != nil {
		t.Fatalf("evaluate and fire: %v", err)
	}
	if fired != 0 {
		t.Fatalf("expected disabled rule not to fire, got %d", fired)
	}
}
