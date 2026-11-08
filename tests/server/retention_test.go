package test

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/storage"
)

func TestRunRetentionJobDeletesOnTick(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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
