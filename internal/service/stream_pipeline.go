package service

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
)

// StreamIngestPipeline validates and dedups events synchronously (the same
// correctness checks IngestService makes), then publishes accepted events
// onto a StreamProducer instead of inserting them directly. A consumer
// worker (started elsewhere by calling a StreamConsumer's Start with
// p.Consume as the handler) performs the actual DB insert, session bump, and
// materialized aggregate update — decoupling the ingest request path from
// storage, which is Phase 7's goal.
type StreamIngestPipeline struct {
	Repo *repository.Repo
	DB   *gorm.DB
	// SetTenantContext scopes the consumer transaction for PostgreSQL RLS.
	// It is nil for SQLite/dev deployments.
	SetTenantContext func(*gorm.DB, string) error
	Producer         StreamProducer
	// TierLimiter, if set, applies the same per-tenant tier limit as the
	// synchronous ingest path after the session resolves the tenant.
	TierLimiter TenantLimiter
	// Sessions, if set, broadcasts per-session updates (EventCount, LastSeen)
	// to subscribers via the LiveSessionHub. Nil means no live-session
	// tracking, so existing callers/tests that don't set it are unaffected.
	Sessions *LiveSessionHub
}

// Ingest has the same accepted/rejected contract as IngestService.Ingest,
// but queues accepted events for async processing instead of writing them
// inline.
func (p *StreamIngestPipeline) Ingest(events []map[string]any) (int, int) {
	accepted, rejected := 0, 0
	for _, e := range events {
		sid, _ := e["session_id"].(string)
		eid, _ := e["event_id"].(string)
		name, nameOK := e["event_name"].(string)
		ts, tsOK := e["client_ts"].(float64)
		eventVersion := 1
		if rawVersion, ok := e["event_version"].(float64); ok {
			eventVersion = int(rawVersion)
		}
		if sid == "" || eid == "" || !nameOK || name == "" || !tsOK || eventVersion < 1 {
			rejected++
			continue
		}
		if p.Repo.EventExists(sid, eid) {
			rejected++
			continue
		}
		sess, err := p.Repo.GetSession(sid)
		if err != nil {
			rejected++
			continue
		}
		if p.TierLimiter != nil {
			tier := ""
			if tenant, terr := p.Repo.GetTenant(sess.TenantID); terr == nil {
				tier = tenant.Tier
			}
			if !p.TierLimiter.Allow(sess.TenantID, tier) {
				rejected++
				continue
			}
		}
		payload, _ := json.Marshal(e["payload"])
		dims, _ := json.Marshal(e["dims"])
		source, _ := e["source"].(string)
		if source == "" {
			source = "client"
		}
		ev := &model.Event{
			EventID: eid, SessionID: sid, TenantID: sess.TenantID,
			EventName: name, ClientTS: int64(ts), EventVersion: eventVersion, IngestTS: time.Now(),
			Payload: string(payload), Dims: string(dims), Source: source,
		}
		ev.ContentHash = eventContentHash(ev.TenantID, ev.SessionID, ev.EventName, ev.ClientTS, payload, dims)
		if source != "replay" && p.Repo.EventContentExists(ev.ContentHash) {
			rejected++
			continue
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
	if p.SetTenantContext == nil {
		if err := p.Repo.InsertEvent(ev); err != nil {
			return err
		}
		if sess, err := p.Repo.GetSession(ev.SessionID); err == nil {
			sess.EventCount++
			sess.LastSeen = time.Now()
			_ = p.Repo.UpdateSession(sess)
			if p.Sessions != nil {
				p.Sessions.Update(sess)
				p.Sessions.AddEvent(ev.SessionID, ev)
			}
		}
		return UpsertAggregateCount(p.DB, ev.TenantID, ev.EventName, ev.Dims)
	}

	tx := p.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := p.SetTenantContext(tx, ev.TenantID); err != nil {
		tx.Rollback()
		return err
	}
	txRepo := repository.New(tx)
	if err := txRepo.InsertEvent(ev); err != nil {
		tx.Rollback()
		return err
	}
	var session *model.Session
	if sess, err := txRepo.GetSession(ev.SessionID); err == nil {
		sess.EventCount++
		sess.LastSeen = time.Now()
		if err := txRepo.UpdateSession(sess); err != nil {
			tx.Rollback()
			return err
		}
		session = sess
	}
	if err := UpsertAggregateCount(tx, ev.TenantID, ev.EventName, ev.Dims); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	if p.Sessions != nil && session != nil {
		p.Sessions.Update(session)
		p.Sessions.AddEvent(ev.SessionID, ev)
	}
	return nil
}
