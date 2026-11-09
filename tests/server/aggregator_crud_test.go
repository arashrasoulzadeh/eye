package test

import (
	"testing"

	"github.com/eye/eye/internal/service"
)

func TestAggregatorServiceGetUpdateDelete(t *testing.T) {
	db := newAggregatorTestDB(t)
	svc := &service.AggregatorService{DB: db}

	agg, err := svc.Create("high_value", "t1", `{"event_name":"purchase"}`, "high_value_purchase", 5, "1h")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := svc.Get(agg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "high_value" {
		t.Fatalf("get returned wrong aggregation: %+v", got)
	}

	updated, err := svc.Update(agg.ID, "high_value_v2", `{"event_name":"purchase"}`, "high_value_purchase", 10, "1d")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "high_value_v2" || updated.Threshold != 10 || updated.Window != "1d" {
		t.Fatalf("update did not persist changes: %+v", updated)
	}
	if updated.UpdatedAt < updated.CreatedAt {
		t.Error("UpdatedAt should not be before CreatedAt")
	}

	if err := svc.Delete(agg.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(agg.ID); err == nil {
		t.Fatal("expected Get to fail after Delete")
	}
}

func TestAggregatorServiceGetMissing(t *testing.T) {
	db := newAggregatorTestDB(t)
	svc := &service.AggregatorService{DB: db}
	if _, err := svc.Get("does-not-exist"); err == nil {
		t.Fatal("expected error for missing aggregation")
	}
}
