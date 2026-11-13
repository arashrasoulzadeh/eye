package service

import (
	"context"
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunIntelligenceEvaluationJobPersistsEvaluation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.IntelligenceModel{}, &model.IntelligenceEvaluation{}, &model.UserIntelligence{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.Create(&model.IntelligenceModel{ID: "t1:behavior:1", TenantID: "t1", Label: "behavior", Version: 1, Algorithm: "test", Metrics: `{"segments":{"new_user":1}}`, Status: "candidate", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go RunIntelligenceEvaluationJob(ctx, &UserIntelligenceService{DB: db}, time.Hour, nil)
	time.Sleep(10 * time.Millisecond)
	cancel()
	var evaluation model.IntelligenceEvaluation
	if err := db.First(&evaluation).Error; err != nil {
		t.Fatal(err)
	}
	if evaluation.TenantID != "t1" || evaluation.Status == "" {
		t.Fatalf("unexpected evaluation: %#v", evaluation)
	}
}

func TestIntelligenceModelNeedsRetrainingWhenStale(t *testing.T) {
	now := time.Now()
	if !intelligenceModelNeedsRetraining(model.IntelligenceModel{ID: "m1", CreatedAt: now.Add(-intelligenceRetrainAfter)}, now) {
		t.Fatal("stale model was not marked for retraining")
	}
	if intelligenceModelNeedsRetraining(model.IntelligenceModel{ID: "m2", CreatedAt: now.Add(-time.Minute)}, now) {
		t.Fatal("fresh model was marked for retraining")
	}
	if !intelligenceModelNeedsRetraining(model.IntelligenceModel{}, now) {
		t.Fatal("missing model was not marked for retraining")
	}
}
