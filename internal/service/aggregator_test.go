package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func newTestDB_aggregator_test(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Aggregation{}, &model.AggregateCount{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestRunAggregationJob(t *testing.T) {
	db := newTestDB_aggregator_test(t)
	svc := &AggregatorService{DB: db}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	RunAggregationJob(ctx, svc, 10*time.Millisecond, nil)
	// Should complete without error
}

func TestAggregatorListPageFiltersAndOrders(t *testing.T) {
	db := newTestDB_aggregator_test(t)
	now := time.Now().Unix()
	db.Create(&model.Aggregation{ID: "old", TenantID: "t1", Name: "old clicks", Rule: "{}", UpdatedAt: now - 10})
	db.Create(&model.Aggregation{ID: "new", TenantID: "t1", Name: "new clicks", Rule: "{}", UpdatedAt: now})
	db.Create(&model.Aggregation{ID: "other", TenantID: "t1", Name: "purchases", Rule: "{}", UpdatedAt: now + 10})

	items, total, err := (&AggregatorService{DB: db}).ListPage("t1", "clicks", "desc", 1, 0)
	if err != nil || total != 2 || len(items) != 1 || items[0].ID != "new" {
		t.Fatalf("unexpected page: items=%+v total=%d err=%v", items, total, err)
	}
}
