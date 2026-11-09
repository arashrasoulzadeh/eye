package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func newTestDB_experiment_test(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Experiment{}, &model.ExperimentAssignment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestExperimentService_ActiveAssignmentForSession(t *testing.T) {
	db := newTestDB_experiment_test(t)
	svc := &ExperimentService{DB: db}

	// Create experiment and assignment
	db.Create(&model.Experiment{
		ID:       "exp1",
		TenantID: "t1",
		Name:     "test",
		Variants: `["a","b"]`,
		Traffic:  100,
		Enabled:  true,
	})
	db.Create(&model.ExperimentAssignment{
		ExperimentID: "exp1",
		SessionID:    "s1",
		Variant:      "a",
		AssignedAt:   time.Now().UnixMilli(),
	})

	expID, variant, ok := svc.ActiveAssignmentForSession("s1")
	if !ok {
		t.Fatal("expected assignment found")
	}
	if expID != "exp1" || variant != "a" {
		t.Errorf("got %s/%s, want exp1/a", expID, variant)
	}

	// Non-existent session
	_, _, ok = svc.ActiveAssignmentForSession("nonexistent")
	if ok {
		t.Error("expected false for nonexistent session")
	}
}