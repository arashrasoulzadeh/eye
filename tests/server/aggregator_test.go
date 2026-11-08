package test

import (
	"encoding/json"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/service"
)

func newAggregatorTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := db.AutoMigrate(&model.Aggregation{}, &model.Event{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAggregatorServiceCreateAndList(t *testing.T) {
	db := newAggregatorTestDB(t)
	svc := &service.AggregatorService{DB: db}

	agg, err := svc.Create("high_value_purchases", "t1", `{"event_name":"purchase"}`, "high_value_purchase", 5, "1h")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if agg.ID == "" {
		t.Fatal("expected a generated ID")
	}
	if !agg.Enabled {
		t.Error("new aggregation should default to enabled")
	}

	// A different tenant's aggregation must not leak into t1's list.
	if _, err := svc.Create("other", "t2", `{}`, "other_out", 1, "1m"); err != nil {
		t.Fatalf("create for t2: %v", err)
	}

	list, err := svc.List("t1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Name != "high_value_purchases" {
		t.Fatalf("expected exactly t1's aggregation, got %+v", list)
	}
}

func TestAggregatorServiceEmitSystemEvent(t *testing.T) {
	db := newAggregatorTestDB(t)
	svc := &service.AggregatorService{DB: db}

	if err := svc.EmitSystemEvent("t1", "rollup_complete", map[string]any{"count": 42}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	var ev model.Event
	if err := db.Where("tenant_id = ? AND event_name = ?", "t1", "rollup_complete").First(&ev).Error; err != nil {
		t.Fatalf("expected emitted event to be stored: %v", err)
	}
	if ev.Source != "system" || !ev.IsAggregated {
		t.Errorf("expected a system+aggregated event, got %+v", ev)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(ev.Payload), &payload); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if payload["count"].(float64) != 42 {
		t.Errorf("payload count = %v, want 42", payload["count"])
	}
}
