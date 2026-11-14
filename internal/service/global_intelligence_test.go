package service

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGlobalTrainingCSVAndInference(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Event{}, &model.Tenant{}, &model.IntelligenceInference{}, &model.IntelligenceInferenceRun{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, row := range []any{
		&model.Tenant{ID: "tenant-secret"},
		&model.Session{ID: "session-secret", TenantID: "tenant-secret", UserID: "user-secret", StartedAt: now.Add(-time.Hour), LastSeen: now.Add(-time.Hour)},
		&model.Event{EventID: "event-secret", TenantID: "tenant-secret", SessionID: "session-secret", EventName: "question_answered", Payload: "secret payload", ClientTS: now.Add(-time.Hour).UnixMilli()},
		&model.Event{EventID: "purchase-secret", TenantID: "tenant-secret", SessionID: "session-secret", EventName: "purchase", ClientTS: now.Add(-time.Hour + time.Second).UnixMilli()},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	s := &UserIntelligenceService{DB: db}
	var output bytes.Buffer
	if err := s.ExportTrainingCSV(&output, "", now); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil || len(rows) != 2 || rows[1][1] != "question_answered|purchase_confirmed" || strings.Contains(output.String(), "secret") {
		t.Fatalf("%v %v", rows, err)
	}
	bundle := GlobalIntelligenceBundle{Schema: 1, Version: "model-test", Next: map[string]string{"question_answered": "purchase_confirmed"}}
	if err := s.ScoreGlobalModel(t.Context(), bundle, now); err != nil {
		t.Fatal(err)
	}
	var result model.IntelligenceInference
	if err := db.First(&result, "kind = ? AND entity_id = ?", "user", "user-secret").Error; err != nil || !strings.Contains(result.Tags, "buyer") || result.Version != bundle.Version {
		t.Fatalf("%+v %v", result, err)
	}
	bundle.Version = "model-next"
	if err := s.ScoreGlobalModel(t.Context(), bundle, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var stale int64
	if err := db.Model(&model.IntelligenceInference{}).Where("version <> ?", bundle.Version).Count(&stale).Error; err != nil || stale != 0 {
		t.Fatalf("stale inference rows=%d err=%v", stale, err)
	}

}

func TestGlobalTagCatalogContains200Keys(t *testing.T) {
	seen := map[string]bool{}
	for _, tags := range GlobalTagCatalog {
		for _, tag := range tags {
			if seen[tag] {
				t.Fatalf("duplicate tag %q", tag)
			}
			seen[tag] = true
		}
	}
	if len(seen) != 200 {
		t.Fatalf("got %d tag keys, want 200", len(seen))
	}
}
