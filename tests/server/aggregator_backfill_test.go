package test

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func seedPurchaseEvents(t *testing.T, db *gorm.DB, tenantID string, n int, sessionPrefix string) {
	for i := 0; i < n; i++ {
		ev := &model.Event{
			EventID:   sessionPrefix + "-ev-" + time.Now().Add(time.Duration(i)*time.Millisecond).Format("150405.000000000"),
			SessionID: sessionPrefix,
			TenantID:  tenantID,
			EventName: "purchase",
			ClientTS:  time.Now().UnixMilli(),
			IngestTS:  time.Now(),
			Payload:   "{}",
		}
		if err := db.Create(ev).Error; err != nil {
			t.Fatalf("seed event: %v", err)
		}
	}
}

func TestAggregatorServiceBackfillEmitsOnThreshold(t *testing.T) {
	db := newAggregatorTestDB(t)
	svc := &service.AggregatorService{DB: db}

	agg, err := svc.Create("high_value", "t1", `{"event_name":"purchase"}`, "high_value_purchase", 3, "1h")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	seedPurchaseEvents(t, db, "t1", 5, "s1")

	emitted, err := svc.Backfill(agg.ID)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if emitted != 1 {
		t.Fatalf("expected 1 rollup emitted (5 >= threshold 3), got %d", emitted)
	}

	var out model.Event
	if err := db.Where("tenant_id = ? AND event_name = ?", "t1", "high_value_purchase").First(&out).Error; err != nil {
		t.Fatalf("expected output event to exist: %v", err)
	}
	if !out.IsAggregated || out.Source != "system" {
		t.Errorf("output event should be a system aggregated event, got %+v", out)
	}
}

func TestRefreshMaterializedCountsRebuildsTenantReadModel(t *testing.T) {
	db := newAggregatorTestDB(t)
	if err := db.AutoMigrate(&model.AggregateCount{}); err != nil {
		t.Fatal(err)
	}
	svc := &service.AggregatorService{DB: db}
	db.Create(&model.Event{EventID: "m1", TenantID: "t1", EventName: "click", Dims: "{}"})
	db.Create(&model.Event{EventID: "m2", TenantID: "t1", EventName: "click", Dims: "{}"})
	db.Create(&model.Event{EventID: "m3", TenantID: "t2", EventName: "click", Dims: "{}"})
	db.Create(&model.AggregateCount{TenantID: "t1", EventName: "click", DimsKey: "{}", Count: 99})
	db.Create(&model.AggregateCount{TenantID: "t2", EventName: "click", DimsKey: "{}", Count: 7})

	if err := svc.RefreshMaterializedCounts("t1"); err != nil {
		t.Fatal(err)
	}
	var count model.AggregateCount
	if err := db.First(&count, "tenant_id = ? AND event_name = ? AND dims_key = ?", "t1", "click", "{}").Error; err != nil {
		t.Fatal(err)
	}
	if count.Count != 2 {
		t.Fatalf("expected rebuilt count 2, got %d", count.Count)
	}
	var other model.AggregateCount
	if err := db.First(&other, "tenant_id = ?", "t2").Error; err != nil {
		t.Fatal(err)
	}
}

func TestAggregatorServiceBackfillSkipsBelowThreshold(t *testing.T) {
	db := newAggregatorTestDB(t)
	svc := &service.AggregatorService{DB: db}

	agg, err := svc.Create("high_value", "t1", `{"event_name":"purchase"}`, "high_value_purchase", 10, "1h")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	seedPurchaseEvents(t, db, "t1", 2, "s1")

	emitted, err := svc.Backfill(agg.ID)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("expected no rollup below threshold, got %d emitted", emitted)
	}
}

func TestAggregatorServiceBackfillIsIdempotent(t *testing.T) {
	db := newAggregatorTestDB(t)
	svc := &service.AggregatorService{DB: db}

	agg, err := svc.Create("high_value", "t1", `{"event_name":"purchase"}`, "high_value_purchase", 3, "1h")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	seedPurchaseEvents(t, db, "t1", 5, "s1")

	first, err := svc.Backfill(agg.ID)
	if err != nil {
		t.Fatalf("first backfill: %v", err)
	}
	if first != 1 {
		t.Fatalf("expected 1 emitted on first run, got %d", first)
	}

	second, err := svc.Backfill(agg.ID)
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if second != 0 {
		t.Fatalf("expected 0 emitted on second run (already-aggregated events must not be recounted), got %d", second)
	}

	var count int64
	db.Model(&model.Event{}).Where("tenant_id = ? AND event_name = ?", "t1", "high_value_purchase").Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 output event after two backfill runs, got %d", count)
	}
}

func TestAggregatorServiceBackfillPicksUpNewEventsAfterRerun(t *testing.T) {
	db := newAggregatorTestDB(t)
	svc := &service.AggregatorService{DB: db}

	agg, err := svc.Create("high_value", "t1", `{"event_name":"purchase"}`, "high_value_purchase", 3, "1h")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	seedPurchaseEvents(t, db, "t1", 3, "s1")
	if _, err := svc.Backfill(agg.ID); err != nil {
		t.Fatalf("first backfill: %v", err)
	}

	// New, not-yet-aggregated events for the same session arrive later.
	seedPurchaseEvents(t, db, "t1", 3, "s1")

	emitted, err := svc.Backfill(agg.ID)
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if emitted != 1 {
		t.Fatalf("expected the new batch to trigger a second rollup, got %d", emitted)
	}

	var count int64
	db.Model(&model.Event{}).Where("tenant_id = ? AND event_name = ?", "t1", "high_value_purchase").Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 output events total, got %d", count)
	}
}
