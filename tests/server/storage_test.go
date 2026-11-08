package test

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/storage"
)

func newTestRepo(t *testing.T) storage.Repository {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := storage.NewTimescaleRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestStorageRepositoryCreateAndGetSession(t *testing.T) {
	repo := newTestRepo(t)
	s := &model.Session{ID: "s1", TenantID: "t1"}
	if err := repo.CreateSession(s); err != nil {
		t.Fatalf("create session: %v", err)
	}
	got, err := repo.GetSession("s1")
	if err != nil || got.TenantID != "t1" {
		t.Fatalf("get session mismatch: %v %+v", err, got)
	}
}

func TestStorageRepositoryInsertAndGetEvents(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.CreateSession(&model.Session{ID: "s1", TenantID: "t1"}); err != nil {
		t.Fatal(err)
	}
	events := []*model.Event{
		{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click", IngestTS: time.Now()},
		{EventID: "e2", SessionID: "s1", TenantID: "t1", EventName: "view", IngestTS: time.Now()},
	}
	if err := repo.InsertEvents(events); err != nil {
		t.Fatalf("insert events: %v", err)
	}
	got, err := repo.GetEvents("t1", "s1", 10)
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
}

func TestStorageRetentionJobDeletesExpiredEvents(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.CreateSession(&model.Session{ID: "s1", TenantID: "t1"}); err != nil {
		t.Fatal(err)
	}
	old := &model.Event{EventID: "old", SessionID: "s1", TenantID: "t1", EventName: "x", IngestTS: time.Now().Add(-48 * time.Hour)}
	fresh := &model.Event{EventID: "new", SessionID: "s1", TenantID: "t1", EventName: "x", IngestTS: time.Now()}
	if err := repo.InsertEvents([]*model.Event{old, fresh}); err != nil {
		t.Fatal(err)
	}

	deleted, err := repo.DeleteExpiredEvents(24 * time.Hour)
	if err != nil {
		t.Fatalf("retention run: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 expired event deleted, got %d", deleted)
	}

	got, err := repo.GetEvents("t1", "s1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].EventID != "new" {
		t.Fatalf("expected only the fresh event to remain, got %+v", got)
	}
}
