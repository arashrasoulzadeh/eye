package service

import (
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUsageServiceCurrentCountsCalendarMonth(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Event{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	db.Create(&model.Event{EventID: "in", TenantID: "t1", IngestTS: now, Payload: "123", Dims: "{}"})
	db.Create(&model.Event{EventID: "old", TenantID: "t1", IngestTS: now.AddDate(0, -1, 0), Payload: "ignored", Dims: "{}"})
	db.Create(&model.Event{EventID: "other", TenantID: "t2", IngestTS: now, Payload: "ignored", Dims: "{}"})
	usage, err := (&UsageService{DB: db}).Current("t1", now)
	if err != nil {
		t.Fatal(err)
	}
	if usage.EventCount != 1 || usage.StorageBytes != 5 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestUsageServiceCheckEnforcesConfiguredLimits(t *testing.T) {
	s := UsageSnapshot{EventCount: 9, StorageBytes: 90}
	if !((&UsageService{}).Check(s, 10, 100, 10)).Allowed {
		t.Fatal("expected event within both quotas to be allowed")
	}
	if ((&UsageService{}).Check(s, 9, 0, 0)).Allowed {
		t.Fatal("expected event quota to reject the next event")
	}
	if ((&UsageService{}).Check(s, 0, 99, 10)).Allowed {
		t.Fatal("expected storage quota to reject the next payload")
	}
	if !((&UsageService{}).Check(s, 0, 0, 999999)).Allowed {
		t.Fatal("zero limits should mean unlimited")
	}
}
