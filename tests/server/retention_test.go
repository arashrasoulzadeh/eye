package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/storage"
)

func TestRunRetentionJobDeletesOnTick(t *testing.T) {
	// A plain ":memory:" DSN gives each pooled connection its own empty DB,
	// which breaks once the driver opens more than one connection; a bare
	// "cache=shared" DSN fixes that but shares the DB by name across the
	// whole process, colliding with other runs of this same test (e.g.
	// `-count=N`, parallel packages). Scope the shared cache to a unique
	// name per run instead.
	dsn := fmt.Sprintf("file:retention_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, _ := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	repo := storage.NewTimescaleRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(&model.Session{ID: "s1", TenantID: "t1"}); err != nil {
		t.Fatal(err)
	}
	old := &model.Event{EventID: "old", SessionID: "s1", TenantID: "t1", EventName: "x", IngestTS: time.Now().Add(-48 * time.Hour)}
	if err := repo.InsertEvents([]*model.Event{old}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go storage.RunRetentionJob(ctx, repo, 24*time.Hour, 10*time.Millisecond, nil)
	defer cancel()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		events, err := repo.GetEvents("t1", "s1", 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("expected retention job to delete the expired event within the deadline")
}
