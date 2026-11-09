package service

import (
	"encoding/json"
	"time"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/repository"
)

// ExperimentAssigner resolves a session's sticky variant for any experiments
// it is tagged into. It's an interface (not *ExperimentService directly) so
// IngestService doesn't depend on experiment internals, and because not every
// deployment of IngestService needs experiment tagging.
type ExperimentAssigner interface {
	ActiveAssignmentForSession(sessionID string) (experimentID, variant string, ok bool)
}

// TenantLimiter is satisfied by *TenantRateLimiter. Nil (the zero value)
// means no per-tenant tier limiting, so existing callers/tests that don't
// set it are unaffected.
type TenantLimiter interface {
	Allow(tenantID, tier string) bool
}

type IngestService struct {
	Repo   *repository.Repo
	ExpSvc ExperimentAssigner
	// TierLimiter, if set, is consulted once per event after its tenant is
	// resolved from the session, gating ingestion by the tenant's Tier
	// (see TenantRateLimiter). A denied event counts toward "rejected".
	TierLimiter TenantLimiter
	// Counters, if set, is bumped once per accepted event for the event's
	// tenant, feeding the LiveCounters real-time gRPC stream (see
	// live_counters.go). Nil means no live-counter tracking, so existing
	// callers/tests that don't set it are unaffected.
	Counters *LiveCounterHub
}

func (s *IngestService) Ingest(events []map[string]any) (int, int) {
	accepted, rejected := 0, 0
	for _, e := range events {
		sid, _ := e["session_id"].(string)
		eid, _ := e["event_id"].(string)
		eventName, nameOK := e["event_name"].(string)
		clientTS, tsOK := e["client_ts"].(float64)
		// Malformed input (missing/wrong-typed required fields) is rejected,
		// not a panic: a buggy or malicious client should never be able to
		// crash the whole server process over one bad event in a batch.
		if sid == "" || eid == "" || !nameOK || !tsOK {
			rejected++
			continue
		}
		if s.Repo.EventExists(sid, eid) { rejected++; continue }
		payload, _ := json.Marshal(e["payload"])
		dims, _ := json.Marshal(e["dims"])
		source, _ := e["source"].(string)
		if source == "" {
			source = "client"
		}
		ev := &model.Event{
			EventID: eid, SessionID: sid, EventName: eventName,
			ClientTS: int64(clientTS), IngestTS: time.Now(),
			Payload: string(payload), Dims: string(dims),
			Source: source, IsAggregated: false,
		}
		if s.ExpSvc != nil {
			if expID, variant, ok := s.ExpSvc.ActiveAssignmentForSession(sid); ok {
				ev.ExperimentID = expID
				ev.Variant = variant
			}
		}
		// resolve tenant from session
		sess, err := s.Repo.GetSession(sid)
		if err != nil { rejected++; continue }
		ev.TenantID = sess.TenantID
		if s.TierLimiter != nil {
			tier := ""
			if tenant, terr := s.Repo.GetTenant(sess.TenantID); terr == nil {
				tier = tenant.Tier
			}
			if !s.TierLimiter.Allow(sess.TenantID, tier) {
				rejected++
				continue
			}
		}
		if err := s.Repo.InsertEvent(ev); err != nil { rejected++; continue }
		sess.EventCount++
		s.Repo.UpdateSession(sess)
		accepted++
		if s.Counters != nil {
			s.Counters.Increment(ev.TenantID)
		}
	}
	return accepted, rejected
}