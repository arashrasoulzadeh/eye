package test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/service"
)

func TestLocalFSSnapshotStorePutListGetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := &service.LocalFSSnapshotStore{Dir: dir}
	ctx := context.Background()

	if err := store.Put(ctx, "a.db", strings.NewReader("hello")); err != nil {
		t.Fatalf("put: %v", err)
	}

	names, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 1 || names[0] != "a.db" {
		t.Fatalf("expected [a.db], got %v", names)
	}

	// No leftover .tmp files should ever be visible via List.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("found leftover tmp file %q", e.Name())
		}
	}

	r, err := store.Get(ctx, "a.db")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer r.Close()
	buf := make([]byte, 5)
	if _, err := r.Read(buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf) != "hello" {
		t.Fatalf("expected hello, got %q", buf)
	}
}

func TestSnapshotServiceTakeSnapshotIsRestorableAndMatchesData(t *testing.T) {
	// A real on-disk SQLite DB (VACUUM INTO needs a real file, not :memory:).
	dbPath := filepath.Join(t.TempDir(), "source.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Tenant{}, &model.Session{}, &model.Event{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Tenant{ID: "t1", Name: "Acme"})
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click"})
	db.Create(&model.Event{EventID: "e2", SessionID: "s1", TenantID: "t1", EventName: "click"})

	snapshotDir := t.TempDir()
	store := &service.LocalFSSnapshotStore{Dir: snapshotDir}
	snapSvc := &service.SnapshotService{DB: db, DBPath: dbPath, Store: store}

	ctx := context.Background()
	name, err := snapSvc.TakeSnapshot(ctx)
	if err != nil {
		t.Fatalf("take snapshot: %v", err)
	}
	if name == "" {
		t.Fatal("expected a non-empty snapshot name")
	}

	names, err := store.List(ctx)
	if err != nil || len(names) != 1 {
		t.Fatalf("expected exactly one stored snapshot, got %v, err %v", names, err)
	}

	// Restore drill: copy the snapshot into a fresh file and verify the
	// restored database has exactly the same rows as the source did at
	// snapshot time.
	restoredPath := filepath.Join(t.TempDir(), "restored.db")
	if err := service.Restore(ctx, store, name, restoredPath); err != nil {
		t.Fatalf("restore: %v", err)
	}

	restoredDB, err := gorm.Open(sqlite.Open(restoredPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open restored db: %v", err)
	}

	var tenantCount, sessionCount, eventCount int64
	restoredDB.Model(&model.Tenant{}).Count(&tenantCount)
	restoredDB.Model(&model.Session{}).Count(&sessionCount)
	restoredDB.Model(&model.Event{}).Count(&eventCount)

	if tenantCount != 1 || sessionCount != 1 || eventCount != 2 {
		t.Fatalf("restored counts don't match source: tenants=%d sessions=%d events=%d", tenantCount, sessionCount, eventCount)
	}

	var restoredEvent model.Event
	if err := restoredDB.First(&restoredEvent, "event_id = ?", "e1").Error; err != nil {
		t.Fatalf("expected e1 to exist in restored db: %v", err)
	}
	if restoredEvent.EventName != "click" {
		t.Fatalf("restored event data mismatch: %+v", restoredEvent)
	}
}

func TestSnapshotServiceTakeSnapshotTwiceProducesTwoIndependentFiles(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "source.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Tenant{}); err != nil {
		t.Fatal(err)
	}

	snapshotDir := t.TempDir()
	store := &service.LocalFSSnapshotStore{Dir: snapshotDir}
	snapSvc := &service.SnapshotService{DB: db, DBPath: dbPath, Store: store}

	ctx := context.Background()
	n1, err := snapSvc.TakeSnapshot(ctx)
	if err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	n2, err := snapSvc.TakeSnapshot(ctx)
	if err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	if n1 == n2 {
		t.Fatalf("expected distinct snapshot names, got %q twice", n1)
	}

	names, err := store.List(ctx)
	if err != nil || len(names) != 2 {
		t.Fatalf("expected 2 stored snapshots, got %v, err %v", names, err)
	}
}
