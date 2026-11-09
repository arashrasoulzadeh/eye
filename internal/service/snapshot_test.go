package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func newTestDB_snapshot_test(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Tenant{}, &model.Session{}, &model.Event{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestRunSnapshotJob(t *testing.T) {
	db := newTestDB_snapshot_test(t)
	db.Create(&model.Tenant{ID: "t1", Name: "Test"})

	store := &LocalFSSnapshotStore{Dir: t.TempDir()}
	svc := &SnapshotService{DB: db, DBPath: t.TempDir() + "/source.db", Store: store}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	RunSnapshotJob(ctx, svc, 10*time.Millisecond, nil)
	// Should complete without error
}

func TestSnapshotService_Restore(t *testing.T) {
	srcDB := newTestDB_snapshot_test(t)
	srcDB.Create(&model.Tenant{ID: "t1", Name: "Test"})

	store := &LocalFSSnapshotStore{Dir: t.TempDir()}
	svc := &SnapshotService{DB: srcDB, DBPath: t.TempDir() + "/source.db", Store: store}

	name, err := svc.TakeSnapshot(context.Background())
	if err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}

	restoredPath := t.TempDir() + "/restored.db"
	if err := Restore(context.Background(), store, name, restoredPath); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	restoredDB, err := gorm.Open(sqlite.Open(restoredPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open restored: %v", err)
	}

	var count int64
	restoredDB.Model(&model.Tenant{}).Count(&count)
	if count != 1 {
		t.Errorf("restored tenant count = %d, want 1", count)
	}
}