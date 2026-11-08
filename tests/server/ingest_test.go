package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
)

func TestIngestDedup(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	r.Migrate()
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	svc := &service.IngestService{Repo: r}
	ev := []map[string]any{{"event_id": "e1", "event_name": "x", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"}}
	a, _ := svc.Ingest(ev)
	if a != 1 { t.Fatalf("expected 1 accepted got %d", a) }
	a2, _ := svc.Ingest(ev)
	if a2 != 0 { t.Fatalf("expected 0 on dup got %d", a2) }
}
