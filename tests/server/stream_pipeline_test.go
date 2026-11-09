package test

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
)

func newPipeline(t *testing.T) (*service.StreamIngestPipeline, *service.ChannelStream, *gorm.DB, func()) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
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
