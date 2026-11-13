package service

import (
	"strings"
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserIntelligenceRefreshTagsBuyerAndBot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Event{}, &model.UserIntelligence{}, &model.IntelligenceModel{}, &model.IntelligenceTrainingRun{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	db.Create(&model.Session{ID: "s1", TenantID: "t1", IP: "1.2.3.4", UserAgent: "GPTBot/1.0", StartedAt: now, LastSeen: now})
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "purchase", ClientTS: now.UnixMilli()})
	if _, err := (&UserIntelligenceService{DB: db}).Refresh("t1", now); err != nil {
		t.Fatal(err)
	}
	rows, err := (&UserIntelligenceService{DB: db}).List("t1", 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	seen := map[string]bool{}
	for _, tag := range rows[0].Tags {
		seen[tag] = true
	}
	if !seen["buyer"] || !seen["bot"] || !seen["ai_crawler"] {
		t.Fatalf("tags=%v", rows[0].Tags)
	}
}

func TestTrainingReadinessRequiresBalancedLabels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.UserIntelligenceLabel{}); err != nil {
		t.Fatal(err)
	}
	svc := &UserIntelligenceService{DB: db}
	for i := 0; i < 5; i++ {
		if err := svc.AddLabel("t1", UserFingerprint("ip", string(rune('a'+i))), "buyer", "test"); err != nil {
			t.Fatal(err)
		}
	}
	ready, err := svc.TrainingReadiness("t1", "buyer")
	if err != nil {
		t.Fatal(err)
	}
	if ready["ready"] != false || ready["positive"] != int64(5) {
		t.Fatalf("unexpected readiness: %#v", ready)
	}
}

func TestUnsupervisedTrainingUsesGlobalBaseAndTenantLayer(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Event{}, &model.UserIntelligence{}, &model.IntelligenceModel{}, &model.IntelligenceTrainingRun{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	db.Create(&model.UserIntelligence{ID: "t1:u1", TenantID: "t1", Fingerprint: "u1", Tags: `["new_user"]`})
	db.Create(&model.UserIntelligence{ID: "t2:u2", TenantID: "t2", Fingerprint: "u2", Tags: `["buyer"]`})
	db.Create(&model.Event{EventID: "e1", TenantID: "t1", SessionID: "s1", EventName: "page_view", IsAggregated: false})
	db.Create(&model.Event{EventID: "e2", TenantID: "t2", SessionID: "s2", EventName: "purchase", IsAggregated: false})
	svc := &UserIntelligenceService{DB: db}
	base, err := svc.TrainUnsupervisedGlobal(now)
	if err != nil || base.TenantID != "" || base.Version != 1 {
		t.Fatalf("global model=%#v err=%v", base, err)
	}
	if !strings.Contains(base.Metrics, `"events":2`) {
		t.Fatalf("global model did not include events: %s", base.Metrics)
	}
	local, err := svc.TrainUnsupervised("t1", now.Add(time.Second))
	if err != nil || local.TenantID != "t1" {
		t.Fatalf("tenant model=%#v err=%v", local, err)
	}
	if !strings.Contains(local.Metrics, `"base_model_version":1`) {
		t.Fatalf("tenant model did not inherit global base: %s", local.Metrics)
	}
	if err := svc.ApplyBehaviorLayers("t1"); err != nil {
		t.Fatal(err)
	}
	var snapshot model.UserIntelligence
	if err := db.First(&snapshot, "id = ?", "t1:u1").Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(snapshot.Scores, "global_behavior") || !strings.Contains(snapshot.Scores, "tenant_behavior") {
		t.Fatalf("layer scores missing: %s", snapshot.Scores)
	}
}

func TestLabelSuggestionsAreExplainableAndDenialsDisappear(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.UserIntelligence{}, &model.UserIntelligenceLabel{}); err != nil {
		t.Fatal(err)
	}
	svc := &UserIntelligenceService{DB: db}
	db.Create(&model.UserIntelligence{ID: "t1:u1", TenantID: "t1", Fingerprint: "u1", IP: "1.2.3.4", UserAgent: "browser", Tags: `["buyer"]`, Scores: `{"buyer":0.95}`})
	suggestions, err := svc.LabelSuggestions("t1", 10)
	if err != nil || len(suggestions) != 1 || suggestions[0].Label != "buyer" || len(suggestions[0].Reasons) == 0 {
		t.Fatalf("suggestions=%#v err=%v", suggestions, err)
	}
	if err := svc.ReviewSuggestion("t1", "u1", "buyer", "denied"); err != nil {
		t.Fatal(err)
	}
	suggestions, err = svc.LabelSuggestions("t1", 10)
	if err != nil || len(suggestions) != 0 {
		t.Fatalf("denied suggestion remained: %#v err=%v", suggestions, err)
	}
}
