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