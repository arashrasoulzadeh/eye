package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func TestSignificanceReportsStrongVariantLift(t *testing.T) {
	result, err := Significance(100, 1000, 180, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Significant95 || result.VariantRate <= result.ControlRate || result.PValue >= 0.05 {
		t.Fatalf("unexpected significance result: %+v", result)
	}
}

func TestSignificanceRejectsInvalidCounts(t *testing.T) {
	if _, err := Significance(2, 1, 0, 10); err == nil {
		t.Fatal("expected invalid counts to be rejected")
	}
}

func TestAutoStopIfWinnerDisablesSignificantWinner(t *testing.T) {
	db := newTestDB_experiment_test(t)
	db.Create(&model.Experiment{ID: "exp-stop", Enabled: true})
	svc := &ExperimentService{DB: db}
	result, err := Significance(100, 1000, 180, 1000)
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := svc.AutoStopIfWinner("exp-stop", result)
	if err != nil || !stopped {
		t.Fatalf("expected experiment to stop, stopped=%v err=%v", stopped, err)
	}
	var exp model.Experiment
	db.First(&exp, "id = ?", "exp-stop")
	if exp.Enabled {
		t.Fatal("expected winning experiment to be disabled")
	}
}

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

func TestExperimentListPageFiltersAndOrders(t *testing.T) {
	db := newTestDB_experiment_test(t)
	db.Create(&model.Experiment{ID: "one", TenantID: "t1", Name: "Checkout", Variants: `["a"]`, CreatedAt: 1})
	db.Create(&model.Experiment{ID: "two", TenantID: "t1", Name: "Signup", Variants: `["b"]`, CreatedAt: 2})
	items, total, err := (&ExperimentService{DB: db}).ListPage("t1", "sign", "desc", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != "two" {
		t.Fatalf("items=%+v total=%d", items, total)
	}
}
