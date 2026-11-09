package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
)

func newTestDB_settings_test(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.RuntimeSettings{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestSettingsService_CurrentRetention(t *testing.T) {
	db := newTestDB_settings_test(t)
	repo := repository.New(db)
	svc := &SettingsService{Repo: repo, DefaultRetentionHours: 720, DefaultRateLimitPerSecond: 50}

	// Test default when no settings in DB
	retention := svc.CurrentRetention()
	if retention != 720*time.Hour {
		t.Errorf("retention = %v, want 720h", retention)
	}

	// Test with DB settings
	db.Create(&model.RuntimeSettings{
		ID:                 1,
		RetentionHours:     168,
		RateLimitPerSecond: 200,
	})

	retention = svc.CurrentRetention()
	if retention != 168*time.Hour {
		t.Errorf("retention = %v, want 168h", retention)
	}
}

func TestSettingsService_CurrentRateLimit(t *testing.T) {
	db := newTestDB_settings_test(t)
	repo := repository.New(db)
	svc := &SettingsService{Repo: repo, DefaultRetentionHours: 720, DefaultRateLimitPerSecond: 50}

	// Test default
	rate := svc.CurrentRateLimit()
	if rate != 50 {
		t.Errorf("rate = %v, want 50", rate)
	}

	// Test with DB settings
	db.Create(&model.RuntimeSettings{
		ID:                 1,
		RetentionHours:     168,
		RateLimitPerSecond: 200,
	})

	rate = svc.CurrentRateLimit()
	if rate != 200 {
		t.Errorf("rate = %v, want 200", rate)
	}
}