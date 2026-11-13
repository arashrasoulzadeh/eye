package service

import (
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func TestDashboardHourBreakdownRespectsRange(t *testing.T) {
	from := time.Date(2026, 11, 13, 10, 37, 0, 0, time.UTC).UnixMilli()
	to := time.Date(2026, 11, 13, 10, 41, 0, 0, time.UTC).UnixMilli()
	events := []model.Event{{ClientTS: time.Date(2026, 11, 13, 10, 39, 0, 0, time.UTC).UnixMilli()}}

	var points []struct {
		Label string `json:"label"`
		Value int64  `json:"value"`
	}
	if err := json.Unmarshal([]byte(dashboardHourBreakdown(events, from, to)), &points); err != nil {
		t.Fatalf("decode chart points: %v", err)
	}
	if len(points) != 1 || points[0].Value != 1 {
		t.Fatalf("unexpected range chart: %#v", points)
	}
}

func newTestDB_dashboard_cache_test(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Tenant{}, &model.Session{}, &model.Event{}, &model.DashboardCache{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestDashboardCacheService_Refresh(t *testing.T) {
	db := newTestDB_dashboard_cache_test(t)
	svc := &DashboardCacheService{DB: db}

	// Setup test data
	db.Create(&model.Tenant{ID: "t1", Name: "Test Tenant"})
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: time.Now().UnixMilli()})

	if err := svc.Refresh(); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	cache, err := svc.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if cache.Events != 1 {
		t.Errorf("Events = %d, want 1", cache.Events)
	}
	if cache.Sessions != 1 {
		t.Errorf("Sessions = %d, want 1", cache.Sessions)
	}
	if cache.Tenants != 1 {
		t.Errorf("Tenants = %d, want 1", cache.Tenants)
	}
	if cache.EventsByHour == "" {
		t.Error("EventsByHour should not be empty")
	}
	if cache.EventsByTenant == "" {
		t.Error("EventsByTenant should not be empty")
	}
}

func TestDashboardCacheService_GetNotFound(t *testing.T) {
	db := newTestDB_dashboard_cache_test(t)
	svc := &DashboardCacheService{DB: db}

	_, err := svc.Get()
	if err == nil {
		t.Error("expected error for missing cache")
	}
}
