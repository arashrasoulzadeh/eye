package service

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/repository"
)

// StreamIngestPipeline validates and dedups events synchronously (the same
// correctness checks IngestService makes), then publishes accepted events
// onto a StreamProducer instead of inserting them directly. A consumer
// worker (started elsewhere by calling a StreamConsumer's Start with
// p.Consume as the handler) performs the actual DB insert, session bump, and
// materialized aggregate update — decoupling the ingest request path from
// storage, which is Phase 7's goal.
type StreamIngestPipeline struct {
	Repo     *repository.Repo
	DB       *gorm.DB
	Producer StreamProducer
}

// Ingest has the same accepted/rejected contract as IngestService.Ingest,
// but queues accepted events for async processing instead of writing them
// inline.
func (p *StreamIngestPipeline) Ingest(events []map[string]any) (int, int) {
	accepted, rejected := 0, 0
	for _, e := range events {
		sid, _ := e["session_id"].(string)
		eid, _ := e["event_id"].(string)
		if p.Repo.EventExists(sid, eid) {
			rejected++
			continue
		}
		sess, err := p.Repo.GetSession(sid)
		if err != nil {
			rejected++
			continue
		}
		name, _ := e["event_name"].(string)
		ts, _ := e["client_ts"].(float64)
		payload, _ := json.Marshal(e["payload"])
		dims, _ := json.Marshal(e["dims"])
		ev := &model.Event{
			EventID: eid, SessionID: sid, TenantID: sess.TenantID,
			EventName: name, ClientTS: int64(ts), IngestTS: time.Now(),
			Payload: string(payload), Dims: string(dims), Source: "client",
		}
		if err := p.Producer.Publish(StreamMessage{Event: ev}); err != nil {
			rejected++
			continue
		}
		accepted++
	}
	return accepted, rejected
}

// Consume is the per-message handler a StreamConsumer worker runs: insert
// the event, bump the session's event count, and fold the event into the
// materialized per-(tenant, event_name, dims) aggregate count.
func (p *StreamIngestPipeline) Consume(msg StreamMessage) error {
	ev := msg.Event
	if err := p.Repo.InsertEvent(ev); err != nil {
		return err
	}
	if sess, err := p.Repo.GetSession(ev.SessionID); err == nil {
		sess.EventCount++
		_ = p.Repo.UpdateSession(sess)
	}
	return UpsertAggregateCount(p.DB, ev.TenantID, ev.EventName, ev.Dims)
}
