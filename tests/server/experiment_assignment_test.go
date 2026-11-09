package test

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func newExperimentTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.New(db).Migrate(); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAssignVariant_StickyAcrossRepeatedCalls(t *testing.T) {
	db := newExperimentTestDB(t)
	svc := &service.ExperimentService{DB: db}
	exp, err := svc.Create("checkout_button_color", "t1", `["control","blue","green"]`, 100)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	first, err := svc.AssignVariant(exp.ID, "session-abc")
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
	if first == "" {
		t.Fatal("expected a non-empty variant")
	}

	for i := 0; i < 5; i++ {
		again, err := svc.AssignVariant(exp.ID, "session-abc")
		if err != nil {
			t.Fatalf("assign repeat %d: %v", i, err)
		}
		if again != first {
			t.Fatalf("assignment not sticky: got %q then %q", first, again)
		}
	}

	var count int64
	db.Model(&model.ExperimentAssignment{}).Where("experiment_id = ? AND session_id = ?", exp.ID, "session-abc").Count(&count)
	if count != 1 {
		t.Errorf("expected exactly one persisted assignment row, got %d", count)
	}
}

func TestAssignVariant_DifferentSessionsCanGetDifferentVariants(t *testing.T) {
	db := newExperimentTestDB(t)
	svc := &service.ExperimentService{DB: db}
	exp, _ := svc.Create("exp", "t1", `["a","b"]`, 100)

	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		v, err := svc.AssignVariant(exp.ID, "session-"+time.Now().Add(time.Duration(i)*time.Second).Format(time.RFC3339Nano))
		if err != nil {
			t.Fatalf("assign: %v", err)
		}
		seen[v] = true
	}
	if len(seen) < 2 {
		t.Errorf("expected both variants to appear across many sessions, saw: %v", seen)
	}
}

func TestAssignVariant_RespectsTrafficSplitProportions(t *testing.T) {
	db := newExperimentTestDB(t)
	svc := &service.ExperimentService{DB: db}
	// Traffic=50 means only ~50% of sessions are included in the experiment at all;
	// included sessions split evenly across the two variants.
	exp, _ := svc.Create("exp", "t1", `["a","b"]`, 50)

	included, excluded := 0, 0
	const n = 2000
	for i := 0; i < n; i++ {
		sid := "sess-" + string(rune('A'+(i%26))) + string(rune('a'+((i/26)%26))) + string(rune('0'+(i%10))) + string(rune('1'+((i*7)%9)))
		v, err := svc.AssignVariant(exp.ID, sid)
		if err != nil {
			t.Fatalf("assign: %v", err)
		}
		if v == "" {
			excluded++
		} else {
			included++
		}
	}
	frac := float64(included) / float64(n)
	if frac < 0.4 || frac > 0.6 {
		t.Errorf("expected roughly 50%% of sessions included, got %.2f (%d/%d)", frac, included, n)
	}
}

func TestAssignVariant_UnknownExperiment(t *testing.T) {
	db := newExperimentTestDB(t)
	svc := &service.ExperimentService{DB: db}
	if _, err := svc.AssignVariant("does-not-exist", "session-x"); err == nil {
		t.Fatal("expected an error for an unknown experiment")
	}
}

func TestIngest_TagsEventsWithStickyExperimentAssignment(t *testing.T) {
	db := newExperimentTestDB(t)
	repo := repository.New(db)
	expSvc := &service.ExperimentService{DB: db}
	ingestSvc := &service.IngestService{Repo: repo, ExpSvc: expSvc}

	tenant := &model.Tenant{ID: "t1", Name: "Acme"}
	db.Create(tenant)
	sess := &model.Session{ID: "sess-1", TenantID: "t1"}
	db.Create(sess)

	exp, err := expSvc.Create("exp1", "t1", `["control","treatment"]`, 100)
	if err != nil {
		t.Fatalf("create experiment: %v", err)
	}
	wantVariant, err := expSvc.AssignVariant(exp.ID, "sess-1")
	if err != nil {
		t.Fatalf("assign: %v", err)
	}

	events := []map[string]any{
		{
			"event_id": "e1", "session_id": "sess-1", "event_name": "click",
			"client_ts": float64(time.Now().UnixMilli()),
			"payload":   map[string]any{}, "dims": map[string]any{},
		},
	}
	accepted, rejected := ingestSvc.Ingest(events)
	if accepted != 1 || rejected != 0 {
		t.Fatalf("expected 1 accepted, 0 rejected, got %d/%d", accepted, rejected)
	}

	var stored model.Event
	if err := db.First(&stored, "event_id = ?", "e1").Error; err != nil {
		t.Fatalf("expected event to be persisted: %v", err)
	}
	if stored.ExperimentID != exp.ID {
		t.Errorf("expected ExperimentID %q, got %q", exp.ID, stored.ExperimentID)
	}
	if stored.Variant != wantVariant {
		t.Errorf("expected Variant %q, got %q", wantVariant, stored.Variant)
	}
}

func TestIngest_NoExperimentTagWhenNoAssignment(t *testing.T) {
	db := newExperimentTestDB(t)
	repo := repository.New(db)
	expSvc := &service.ExperimentService{DB: db}
	ingestSvc := &service.IngestService{Repo: repo, ExpSvc: expSvc}

	tenant := &model.Tenant{ID: "t1", Name: "Acme"}
	db.Create(tenant)
	sess := &model.Session{ID: "sess-2", TenantID: "t1"}
	db.Create(sess)

	events := []map[string]any{
		{
			"event_id": "e2", "session_id": "sess-2", "event_name": "click",
			"client_ts": float64(time.Now().UnixMilli()),
			"payload":   map[string]any{}, "dims": map[string]any{},
		},
	}
	accepted, _ := ingestSvc.Ingest(events)
	if accepted != 1 {
		t.Fatal("expected event to be accepted")
	}

	var stored model.Event
	db.First(&stored, "event_id = ?", "e2")
	if stored.ExperimentID != "" || stored.Variant != "" {
		t.Errorf("expected no experiment tag, got ExperimentID=%q Variant=%q", stored.ExperimentID, stored.Variant)
	}
}
