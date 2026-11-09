package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
)

func newTestDB_replay_test(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Event{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestReplayService_Replay(t *testing.T) {
	db := newTestDB_replay_test(t)
	repo := repository.New(db)
	// Use IngestService as the ingest implementation
	ingestSvc := &IngestService{Repo: repo}
	svc := &ReplayService{DB: db, Ingest: ingestSvc}

	// Create test data
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click"})

	_, err := svc.Replay("t1", 0, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
}

func TestRunReplayJob(t *testing.T) {
	// Replay doesn't have a periodic job, but we can test the service
}