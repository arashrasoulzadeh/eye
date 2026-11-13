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

func TestRunPipelineCompletesGlobalTenantAndQualityStages(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Event{}, &model.EventBucket{}, &model.UserIntelligence{}, &model.UserIntelligenceLabel{}, &model.IntelligenceModel{}, &model.IntelligenceTrainingRun{}, &model.IntelligenceEvaluation{}, &model.IntelligenceAlert{}, &model.IntelligenceDataQuality{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.Create(&model.Session{ID: "pipeline-session", TenantID: "t1", IP: "127.0.0.1", UserAgent: "browser", StartedAt: now, LastSeen: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Event{EventID: "pipeline-event", TenantID: "t1", SessionID: "pipeline-session", EventName: "page_view", ClientTS: now.UnixMilli()}).Error; err != nil {
		t.Fatal(err)
	}
	result, err := (&UserIntelligenceService{DB: db}).RunPipeline("t1", now)
	if err != nil {
		t.Fatal(err)
	}
	if result.Refreshed != 1 || result.QualityStatus == "" || len(result.Steps) < 9 {
		t.Fatalf("unexpected pipeline result: %#v", result)
	}
	var global, tenant model.IntelligenceModel
	if err := db.Where("tenant_id = ? AND label = ?", "", "behavior").First(&global).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("tenant_id = ? AND label = ?", "t1", "behavior").First(&tenant).Error; err != nil {
		t.Fatal(err)
	}
	if global.Status != "promoted" || tenant.Status != "promoted" {
		t.Fatalf("models not promoted: global=%s tenant=%s", global.Status, tenant.Status)
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
	if err := db.Model(&model.IntelligenceModel{}).Where("tenant_id IN ? AND label = ?", []string{"", "t1"}, "behavior").Update("status", "promoted").Error; err != nil {
		t.Fatal(err)
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

func TestReviewSuggestionAcceptsAndPersistsDecision(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.UserIntelligence{}, &model.UserIntelligenceLabel{}); err != nil {
		t.Fatal(err)
	}
	svc := &UserIntelligenceService{DB: db}
	db.Create(&model.UserIntelligence{ID: "t1:u1", TenantID: "t1", Fingerprint: "u1", Tags: `["buyer"]`, Scores: `{"buyer":0.95}`})

	if err := svc.ReviewSuggestion("t1", "u1", "buyer", "accepted"); err != nil {
		t.Fatal(err)
	}
	var label model.UserIntelligenceLabel
	if err := db.First(&label, "tenant_id = ? AND fingerprint = ? AND label = ?", "t1", "u1", "buyer").Error; err != nil {
		t.Fatal(err)
	}
	if label.Decision != "accepted" || label.Source != "suggestion" {
		t.Fatalf("label=%#v", label)
	}
	suggestions, err := svc.LabelSuggestions("t1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 0 {
		t.Fatalf("accepted suggestion remained: %#v", suggestions)
	}
}

func TestQualityAlertsAreDeduplicatedAndScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.IntelligenceAlert{}); err != nil {
		t.Fatal(err)
	}
	svc := &UserIntelligenceService{DB: db}
	now := time.Now()
	if err := svc.RecordQualityAlert(model.IntelligenceEvaluation{TenantID: "t1", Label: "behavior", DriftScore: 0.8, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := svc.RecordQualityAlert(model.IntelligenceEvaluation{TenantID: "t1", Label: "behavior", DriftScore: 0.9, CreatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	alerts, err := svc.IntelligenceAlerts("t1", "open", 10)
	if err != nil || len(alerts) != 1 {
		t.Fatalf("alerts=%#v err=%v", alerts, err)
	}
	if err := svc.ResolveIntelligenceAlert("t1", alerts[0].ID, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	open, err := svc.IntelligenceAlerts("t1", "open", 10)
	if err != nil || len(open) != 0 {
		t.Fatalf("open=%#v err=%v", open, err)
	}
}

func TestModelsAreReturnedNewestFirstAndTenantScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.IntelligenceModel{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	db.Create(&model.IntelligenceModel{ID: "t1:behavior:1", TenantID: "t1", Label: "behavior", Version: 1, CreatedAt: now.Add(-time.Hour)})
	db.Create(&model.IntelligenceModel{ID: "t1:behavior:2", TenantID: "t1", Label: "behavior", Version: 2, CreatedAt: now})
	db.Create(&model.IntelligenceModel{ID: "t2:behavior:1", TenantID: "t2", Label: "behavior", Version: 1, CreatedAt: now})
	models, err := (&UserIntelligenceService{DB: db}).Models("t1", "behavior", 10)
	if err != nil || len(models) != 2 || models[0].Version != 2 {
		t.Fatalf("models=%#v err=%v", models, err)
	}
}

func TestEvaluationsAreReturnedNewestFirstAndTenantScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.IntelligenceEvaluation{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	db.Create(&model.IntelligenceEvaluation{ID: "t1:1", TenantID: "t1", Label: "behavior", ModelVersion: 1, CreatedAt: now.Add(-time.Hour)})
	db.Create(&model.IntelligenceEvaluation{ID: "t1:2", TenantID: "t1", Label: "behavior", ModelVersion: 2, CreatedAt: now})
	db.Create(&model.IntelligenceEvaluation{ID: "t2:1", TenantID: "t2", Label: "behavior", ModelVersion: 1, CreatedAt: now})
	evaluations, err := (&UserIntelligenceService{DB: db}).Evaluations("t1", "behavior", 10)
	if err != nil || len(evaluations) != 2 || evaluations[0].ModelVersion != 2 {
		t.Fatalf("evaluations=%#v err=%v", evaluations, err)
	}
}

func TestModelLayersIncludeGlobalAndTenantOnly(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.IntelligenceModel{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.IntelligenceModel{ID: "global:behavior:1", TenantID: "", Label: "behavior", Version: 1})
	db.Create(&model.IntelligenceModel{ID: "t1:behavior:1", TenantID: "t1", Label: "behavior", Version: 1})
	db.Create(&model.IntelligenceModel{ID: "t2:behavior:1", TenantID: "t2", Label: "behavior", Version: 1})
	layers, err := (&UserIntelligenceService{DB: db}).ModelLayers("t1", "behavior", 10)
	if err != nil || len(layers) != 2 {
		t.Fatalf("layers=%#v err=%v", layers, err)
	}
	for _, layer := range layers {
		if layer.TenantID != "" && layer.TenantID != "t1" {
			t.Fatalf("unexpected layer=%#v", layer)
		}
	}
}
