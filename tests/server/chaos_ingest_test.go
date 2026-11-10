package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/repository"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func TestIngestRejectsCleanlyWhenDatabaseFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.New(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	svc := &service.IngestService{Repo: repo}
	accepted, rejected := svc.Ingest([]map[string]any{{
		"session_id": "missing",
		"event_id":   "event-1",
		"event_name": "click",
		"client_ts":  float64(1),
	}})
	if accepted != 0 || rejected != 1 {
		t.Fatalf("expected database failure to reject safely, got accepted=%d rejected=%d", accepted, rejected)
	}
}
