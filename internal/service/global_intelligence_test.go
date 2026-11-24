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
	for _, kind := range []string{"user", "session", "event"} {
		id := "session-secret"
		if kind == "event" {
			id = "event-secret"
		}
		tags, err := s.IntelligenceTags(t.Context(), kind, []IntelligenceTagRef{
			{TenantID: "wrong-tenant", ID: id},
			{TenantID: "tenant-secret", ID: id},
		})
		if err != nil || len(tags) != 2 || len(tags[0]) != 0 || len(tags[1]) == 0 {
			t.Fatalf("%s tags=%v err=%v", kind, tags, err)
		}
	}

    // Repeated scoring must preserve live predictions and remove stale entities.
    if err := db.Create(&model.IntelligenceInference{TenantID:"tenant-secret", Kind:"event", EntityID:"removed-event", Version:bundle.Version, ScoredAt:now.Add(-time.Hour)}).Error; err != nil { t.Fatal(err) }
    if err := s.ScoreGlobalModel(t.Context(), bundle, now.Add(time.Second)); err != nil { t.Fatal(err) }
    var count int64
    if err := db.Model(&model.IntelligenceInference{}).Where("tenant_id = ? AND entity_id = ?", "tenant-secret","removed-event").Count(&count).Error; err != nil || count != 0 { t.Fatalf("stale entity count=%d err=%v",count,err) }
    if err := db.Model(&model.IntelligenceInference{}).Where("tenant_id = ? AND entity_id = ?", "tenant-secret","event-secret").Count(&count).Error; err != nil || count != 1 { t.Fatalf("live entity count=%d err=%v",count,err) }

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

func TestScoringInputSignature(t *testing.T) {
 db,err:=gorm.Open(sqlite.Open(":memory:"),&gorm.Config{})
 if err!=nil{t.Fatal(err)}
 if err:=db.AutoMigrate(&model.Session{},&model.Event{});err!=nil{t.Fatal(err)}
 now:=time.Now().UTC()
 session:=model.Session{ID:"s",TenantID:"t",UserID:"u",LastSeen:now}
 if err:=db.Create(&session).Error;err!=nil{t.Fatal(err)}
 bundle:=GlobalIntelligenceBundle{Version:"v1"}
 signature:=func(at time.Time)string{
  t.Helper()
  value,err:=scoringInputSignature(db,"t",bundle,at)
  if err!=nil{t.Fatal(err)}
  return value
 }
 original:=signature(now)
 if signature(now.Add(time.Minute))!=original{t.Fatal("unchanged inputs changed signature")}
 if signature(now.Add(31*time.Minute))==original{t.Fatal("inactivity not detected")}
 event:=model.Event{EventID:"e",TenantID:"t",SessionID:"s",EventName:"purchase",ClientTS:now.Add(-time.Hour).UnixMilli()}
 if err:=db.Create(&event).Error;err!=nil{t.Fatal(err)}
 changed:=signature(now)
 if changed==original{t.Fatal("late event not detected")}
 if err:=db.Delete(&event).Error;err!=nil{t.Fatal(err)}
 if signature(now)!=original{t.Fatal("deletion not detected")}
 bundle.Version="v2"
 if signature(now)==original{t.Fatal("model change not detected")}
}
