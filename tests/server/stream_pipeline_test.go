package test

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

type denyTenantLimiter struct{}

func (denyTenantLimiter) Allow(string, string) bool { return false }

func newPipeline(t *testing.T) (*service.StreamIngestPipeline, *service.ChannelStream, *gorm.DB, func()) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// Pin to a single connection — a plain ":memory:" DSN otherwise hands out a
	// fresh empty DB per pooled connection, which races the consumer goroutine
	// against the test's own queries (see cohort_test.go's newCohortTestDB for
	// the same fix applied there).
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	r := repository.New(db)
	if err := r.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})

	stream := service.NewChannelStream(16)
	p := &service.StreamIngestPipeline{Repo: r, DB: db, Producer: stream}
	ctx, cancel := context.WithCancel(context.Background())
	stream.Start(ctx, p.Consume)
	return p, stream, db, cancel
}

func TestStreamPipelineConsumerInsertsAndAggregates(t *testing.T) {
	p, stream, db, cancel := newPipeline(t)
	defer cancel()

	events := []map[string]any{
		{"event_id": "e1", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{"platform": "web"}, "session_id": "s1"},
	}
	a, rej := p.Ingest(events)
	if a != 1 || rej != 0 {
		t.Fatalf("expected 1 accepted 0 rejected, got %d/%d", a, rej)
	}
	stream.Wait()

	var count int64
	db.Model(&model.Event{}).Where("event_id = ?", "e1").Count(&count)
	if count != 1 {
		t.Fatalf("expected event to be inserted by consumer, found %d", count)
	}

	var sess model.Session
	db.First(&sess, "id = ?", "s1")
	if sess.EventCount != 1 {
		t.Fatalf("expected session event count 1, got %d", sess.EventCount)
	}

	got, err := service.GetAggregateCount(db, "t1", "click", `{"platform":"web"}`)
	if err != nil {
		t.Fatalf("get aggregate count: %v", err)
	}
	if got != 1 {
		t.Fatalf("expected materialized aggregate count 1, got %d", got)
	}
}

func TestStreamPipelineRejectsDuplicateAndUnknownSession(t *testing.T) {
	p, stream, _, cancel := newPipeline(t)
	defer cancel()

	ev := map[string]any{"event_id": "e1", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"}
	a, rej := p.Ingest([]map[string]any{ev})
	if a != 1 || rej != 0 {
		t.Fatalf("first ingest: expected 1/0, got %d/%d", a, rej)
	}
	stream.Wait()

	// Same event_id again — EventExists should now see the consumer's insert
	// and reject it as a duplicate.
	a2, rej2 := p.Ingest([]map[string]any{ev})
	if a2 != 0 || rej2 != 1 {
		t.Fatalf("duplicate ingest: expected 0/1, got %d/%d", a2, rej2)
	}

	unknown := map[string]any{"event_id": "e2", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "no-such-session"}
	a3, rej3 := p.Ingest([]map[string]any{unknown})
	if a3 != 0 || rej3 != 1 {
		t.Fatalf("unknown session ingest: expected 0/1, got %d/%d", a3, rej3)
	}
}

func TestStreamPipelineAppliesTenantRateLimit(t *testing.T) {
	p, stream, db, cancel := newPipeline(t)
	defer cancel()
	p.TierLimiter = denyTenantLimiter{}

	event := map[string]any{"event_id": "limited", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"}
	accepted, rejected := p.Ingest([]map[string]any{event})
	if accepted != 0 || rejected != 1 {
		t.Fatalf("expected rate-limited event to be rejected, got %d/%d", accepted, rejected)
	}
	stream.Wait()

	var count int64
	db.Model(&model.Event{}).Where("event_id = ?", "limited").Count(&count)
	if count != 0 {
		t.Fatalf("rate-limited event was published to the stream")
	}
}

// TestStreamPipelineEndToEndIngestToQuery exercises the full Phase 7 path:
// an event goes in through the async pipeline (producer), is picked up by
// the consumer, inserted into storage, and folded into the materialized
// aggregate — which a caller then reads back ("query"), end to end.
func TestStreamPipelineEndToEndIngestToQuery(t *testing.T) {
	p, stream, db, cancel := newPipeline(t)
	defer cancel()

	for i := 0; i < 3; i++ {
		events := []map[string]any{
			{"event_id": "e" + string(rune('a'+i)), "event_name": "purchase", "client_ts": float64(i), "payload": map[string]any{}, "dims": map[string]any{"platform": "ios"}, "session_id": "s1"},
		}
		a, rej := p.Ingest(events)
		if a != 1 || rej != 0 {
			t.Fatalf("ingest %d: expected 1/0, got %d/%d", i, a, rej)
		}
	}
	stream.Wait()

	got, err := service.GetAggregateCount(db, "t1", "purchase", `{"platform":"ios"}`)
	if err != nil {
		t.Fatalf("get aggregate count: %v", err)
	}
	if got != 3 {
		t.Fatalf("expected materialized aggregate count 3 after 3 events, got %d", got)
	}

	var raw []model.Event
	db.Where("tenant_id = ? AND event_name = ?", "t1", "purchase").Find(&raw)
	if len(raw) != 3 {
		t.Fatalf("expected 3 raw events queryable, got %d", len(raw))
	}
}

func TestStreamPipelineConsumeInsertsEventAndUpdatesSessionAndAggregate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	r := repository.New(db)
	if err := r.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})

	p := &service.StreamIngestPipeline{Repo: r, DB: db}

	msg := service.StreamMessage{
		Event: &model.Event{
			EventID:   "e1",
			SessionID: "s1",
			TenantID:  "t1",
			EventName: "click",
			ClientTS:  1,
			IngestTS:  time.Now(),
			Payload:   "{}",
			Dims:      `{"platform":"web"}`,
			Source:    "client",
		},
	}

	if err := p.Consume(msg); err != nil {
		t.Fatalf("consume: %v", err)
	}

	var count int64
	db.Model(&model.Event{}).Where("event_id = ?", "e1").Count(&count)
	if count != 1 {
		t.Fatalf("expected event to be inserted, found %d", count)
	}

	var sess model.Session
	db.First(&sess, "id = ?", "s1")
	if sess.EventCount != 1 {
		t.Fatalf("expected session event count 1, got %d", sess.EventCount)
	}

	got, err := service.GetAggregateCount(db, "t1", "click", `{"platform":"web"}`)
	if err != nil {
		t.Fatalf("get aggregate count: %v", err)
	}
	if got != 1 {
		t.Fatalf("expected materialized aggregate count 1, got %d", got)
	}
}

func TestUpsertAggregateCountCreatesAndIncrements(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.AggregateCount{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := service.UpsertAggregateCount(db, "t1", "click", `{"platform":"web"}`); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	count, err := service.GetAggregateCount(db, "t1", "click", `{"platform":"web"}`)
	if err != nil || count != 1 {
		t.Fatalf("expected count 1, got %d, err %v", count, err)
	}

	if err := service.UpsertAggregateCount(db, "t1", "click", `{"platform":"web"}`); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	count, err = service.GetAggregateCount(db, "t1", "click", `{"platform":"web"}`)
	if err != nil || count != 2 {
		t.Fatalf("expected count 2, got %d, err %v", count, err)
	}

	// Different dims should be separate
	if err := service.UpsertAggregateCount(db, "t1", "click", `{"platform":"ios"}`); err != nil {
		t.Fatalf("third upsert: %v", err)
	}
	count, err = service.GetAggregateCount(db, "t1", "click", `{"platform":"ios"}`)
	if err != nil || count != 1 {
		t.Fatalf("expected count 1 for ios, got %d, err %v", count, err)
	}
}

func TestGetAggregateCountReturnsZeroForMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.AggregateCount{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	count, err := service.GetAggregateCount(db, "t1", "click", `{"platform":"web"}`)
	if err != nil {
		t.Fatalf("get aggregate count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 for missing, got %d", count)
	}
}
