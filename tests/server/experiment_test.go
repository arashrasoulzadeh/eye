package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/service"
)

func TestExperimentServiceCreate(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := db.AutoMigrate(&model.Experiment{}); err != nil {
		t.Fatal(err)
	}
	svc := &service.ExperimentService{DB: db}

	exp, err := svc.Create("checkout_button_color", "t1", `["control","blue","green"]`, 100)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if exp.ID == "" {
		t.Fatal("expected a generated ID")
	}
	if !exp.Enabled {
		t.Error("new experiment should default to enabled")
	}

	var stored model.Experiment
	if err := db.First(&stored, "id = ?", exp.ID).Error; err != nil {
		t.Fatalf("expected experiment to be persisted: %v", err)
	}
	if stored.Variants != `["control","blue","green"]` || stored.Traffic != 100 {
		t.Errorf("unexpected stored experiment: %+v", stored)
	}
}
