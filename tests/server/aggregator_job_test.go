package test

import (
	"context"
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func TestRunAggregationJobEmitsOnTick(t *testing.T) {
	db := newAggregatorTestDB(t)
	svc := &service.AggregatorService{DB: db}

	agg, err := svc.Create("high_value", "t1", `{"event_name":"purchase"}`, "high_value_purchase", 2, "1h")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go service.RunAggregationJob(ctx, svc, 10*time.Millisecond, nil)
	defer cancel()

	seedPurchaseEvents(t, db, "t1", 2, "s1")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var count int64
		db.Model(&model.Event{}).Where("tenant_id = ? AND event_name = ?", "t1", "high_value_purchase").Count(&count)
		if count == 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = agg
	t.Fatal("expected the background job to roll up the aggregation and emit an output event")
}
