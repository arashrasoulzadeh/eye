package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIngestQueueDrainsTenPercentAndKeepsRemainder(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.PendingStreamEvent{}, &model.Event{}, &model.Session{}, &model.EventBucket{}, &model.AggregateCount{}); err != nil {
		t.Fatal(err)
	}
	queue := &IngestQueue{DB: db}
	events := make([]*model.Event, 101)
	for i := range events {
		events[i] = &model.Event{EventID: fmt.Sprint(i), TenantID: "tenant", EventName: "click"}
	}
	if accepted, err := queue.Enqueue(events, false); err != nil || len(accepted) != 101 {
		t.Fatalf("enqueue accepted %d: %v", len(accepted), err)
	}
	if err := queue.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	var stored, pending int64
	db.Model(&model.Event{}).Count(&stored)
	db.Model(&model.PendingStreamEvent{}).Count(&pending)
	if stored != 11 || pending != 90 {
		t.Fatalf("stored = %d, pending = %d", stored, pending)
	}
}

func TestBothIngestPathsQueueBeforePersisting(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.PendingStreamEvent{}, &model.Event{}, &model.Session{}, &model.EventBucket{}, &model.AggregateCount{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Session{ID: "session", TenantID: "tenant"}).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.New(db)
	queue := &IngestQueue{DB: db}
	ingest := &IngestService{Repo: repo, Queue: queue}
	stream := &StreamIngestPipeline{Repo: repo, Queue: queue}
	first := map[string]any{"event_id": "regular", "session_id": "session", "event_name": "click", "client_ts": float64(1)}
	second := map[string]any{"event_id": "stream", "session_id": "session", "event_name": "click", "client_ts": float64(2)}
	if accepted, rejected := ingest.Ingest([]map[string]any{first}); accepted != 1 || rejected != 0 {
		t.Fatalf("regular ingest accepted %d and rejected %d", accepted, rejected)
	}
	if accepted, rejected := stream.Ingest([]map[string]any{second, first}); accepted != 1 || rejected != 1 {
		t.Fatalf("stream ingest accepted %d and rejected %d", accepted, rejected)
	}
	if repo.EventExists("session", "regular") || repo.EventExists("session", "stream") {
		t.Fatal("event became visible before drain")
	}
	if err := db.Migrator().DropTable(&model.EventBucket{}); err != nil {
		t.Fatal(err)
	}
	if err := queue.Drain(context.Background()); err == nil {
		t.Fatal("expected persistence to fail without the bucket table")
	}
	if repo.EventExists("session", "regular") {
		t.Fatal("failed batch left a persisted event")
	}
	if err := db.AutoMigrate(&model.EventBucket{}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := queue.Drain(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	session, err := repo.GetSession("session")
	if err != nil || session.EventCount != 2 {
		t.Fatalf("session = %+v, error = %v", session, err)
	}
	count, err := GetAggregateCount(db, "tenant", "click", "null")
	if err != nil || count != 1 {
		t.Fatalf("stream aggregate = %d, error = %v", count, err)
	}
}
