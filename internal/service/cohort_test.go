package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func newTestDB_cohort_test(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Cohort{}, &model.Session{}, &model.Event{}, &model.CohortMembership{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestRunCohortMaterializationJob(t *testing.T) {
	db := newTestDB_cohort_test(t)
	svc := &CohortService{DB: db}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	RunCohortMaterializationJob(ctx, svc, 10*time.Millisecond, nil)
	// Should complete without error
}