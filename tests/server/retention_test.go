package test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/storage"
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
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
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
	go storage.RunRetentionJob(ctx, repo, func() time.Duration { return 24 * time.Hour }, 10*time.Millisecond, nil)
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

// TestRunRetentionJobObservesLiveRetentionChange proves retentionFn is
// re-read on every tick: an event that's too young to be deleted under a
// long retention window gets deleted once the caller (standing in for a
// live settings update) shortens the window, without restarting the job.
func TestRunRetentionJobObservesLiveRetentionChange(t *testing.T) {
	dsn := fmt.Sprintf("file:retention_live_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, _ := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	repo := storage.NewTimescaleRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(&model.Session{ID: "s1", TenantID: "t1"}); err != nil {
		t.Fatal(err)
	}
	// 2 hours old: survives a 24h retention window, but not a 1h one.
	e := &model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "x", IngestTS: time.Now().Add(-2 * time.Hour)}
	if err := repo.InsertEvents([]*model.Event{e}); err != nil {
		t.Fatal(err)
	}

	var live atomic.Int64
	live.Store(int64(24 * time.Hour))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go storage.RunRetentionJob(ctx, repo, func() time.Duration { return time.Duration(live.Load()) }, 10*time.Millisecond, nil)

	time.Sleep(100 * time.Millisecond)
	events, err := repo.GetEvents("t1", "s1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected the event to survive the 24h window, got %d events", len(events))
	}

	// Simulate a live settings update shortening retention to 1h.
	live.Store(int64(time.Hour))

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
	t.Fatal("expected the retention job to observe the shortened live retention window and delete the event")
}
