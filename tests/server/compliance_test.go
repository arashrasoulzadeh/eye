package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/service"
)

func newComplianceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Tenant{}, &model.Session{}, &model.Event{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestComplianceServiceEraseSessionDeletesEventsAndSession(t *testing.T) {
	db := newComplianceTestDB(t)
	svc := &service.ComplianceService{DB: db}

	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click"})
	db.Create(&model.Event{EventID: "e2", SessionID: "s1", TenantID: "t1", EventName: "click"})

	// A different session must survive untouched.
	db.Create(&model.Session{ID: "s2", TenantID: "t1"})
	db.Create(&model.Event{EventID: "e3", SessionID: "s2", TenantID: "t1", EventName: "click"})

	deleted, err := svc.EraseSession("s1")
	if err != nil {
		t.Fatalf("erase: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 events deleted, got %d", deleted)
	}

	var sessionCount int64
	db.Model(&model.Session{}).Where("id = ?", "s1").Count(&sessionCount)
	if sessionCount != 0 {
		t.Fatalf("expected session s1 to be gone, found %d", sessionCount)
	}

	var eventCount int64
	db.Model(&model.Event{}).Where("session_id = ?", "s1").Count(&eventCount)
	if eventCount != 0 {
		t.Fatalf("expected s1's events to be gone, found %d", eventCount)
	}

	var otherSession model.Session
	if err := db.First(&otherSession, "id = ?", "s2").Error; err != nil {
		t.Fatalf("expected s2 to survive: %v", err)
	}
	var otherEventCount int64
	db.Model(&model.Event{}).Where("session_id = ?", "s2").Count(&otherEventCount)
	if otherEventCount != 1 {
		t.Fatalf("expected s2's event to survive, found %d", otherEventCount)
	}
}

func TestComplianceServiceEraseUnknownSessionErrors(t *testing.T) {
	db := newComplianceTestDB(t)
	svc := &service.ComplianceService{DB: db}

	_, err := svc.EraseSession("does-not-exist")
	if err != service.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
