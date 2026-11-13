package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
)

func eventContentHash(tenantID, sessionID, name string, clientTS int64, payload, dims []byte) string {
	h := sha256.New()
	for _, part := range []string{tenantID, sessionID, name, fmt.Sprint(clientTS), string(payload), string(dims)} {
		h.Write([]byte{0})
		h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}

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
	Repo *repository.Repo
	// SetTenantContext is provided by the PostgreSQL runtime. It sets the
	// transaction-local RLS tenant; nil keeps SQLite/dev behavior unchanged.
	SetTenantContext func(*gorm.DB, string) error
	ExpSvc           ExperimentAssigner
	// TierLimiter, if set, is consulted once per event after its tenant is
	// resolved from the session, gating ingestion by the tenant's Tier
	// (see TenantRateLimiter). A denied event counts toward "rejected".
	TierLimiter  TenantLimiter
	QuotaLimiter QuotaLimiter
	// Counters, if set, is bumped once per accepted event for the event's
	// tenant, feeding the LiveCounters real-time gRPC stream (see
	// live_counters.go). Nil means no live-counter tracking, so existing
	// callers/tests that don't set it are unaffected.
	Counters *LiveCounterHub
	// Sessions, if set, broadcasts per-session updates (EventCount, LastSeen)
	// to subscribers via the LiveSessionHub. Nil means no live-session
	// tracking, so existing callers/tests that don't set it are unaffected.
	Sessions   *LiveSessionHub
	PIIFields  []string
	DetectPII  bool
	SchemaMode string // off, warn, or reject
}

func (s *IngestService) Ingest(events []map[string]any) (int, int) {
	accepted, rejected := 0, 0
	for _, e := range events {
		sid, _ := e["session_id"].(string)
		eid, _ := e["event_id"].(string)
		eventName, nameOK := e["event_name"].(string)
		clientTS, tsOK := e["client_ts"].(float64)
		eventVersion := 1
		if rawVersion, ok := e["event_version"].(float64); ok {
			eventVersion = int(rawVersion)
		}
		// Malformed input (missing/wrong-typed required fields) is rejected,
		// not a panic: a buggy or malicious client should never be able to
		// crash the whole server process over one bad event in a batch.
		if sid == "" || eid == "" || !nameOK || !tsOK || eventVersion < 1 {
			rejected++
			continue
		}
		if s.Repo.EventExists(sid, eid) {
			rejected++
			continue
		}
		piiFields := s.PIIFields
		if s.DetectPII {
			piiFields = append(append([]string{}, piiFields...), DetectPIIFields(e["payload"])...)
		}
		payload, _ := json.Marshal(RedactPII(e["payload"], piiFields))
		dims, _ := json.Marshal(e["dims"])
		source, _ := e["source"].(string)
		if source == "" {
			source = "client"
		}
		ev := &model.Event{
			EventID: eid, SessionID: sid, EventName: eventName,
			ClientTS: int64(clientTS), EventVersion: eventVersion, IngestTS: time.Now(),
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
		if err != nil {
			rejected++
			continue
		}
		ev.TenantID = sess.TenantID
		if s.SchemaMode == "reject" {
			var schema model.Schema
			if err := s.Repo.DB.Where("tenant_id = ? AND name = ?", ev.TenantID, eventName).Order("version desc").First(&schema).Error; err == nil {
				payloadMap, ok := e["payload"].(map[string]any)
				if !ok {
					rejected++
					continue
				}
				piiFields := s.PIIFields
				if s.DetectPII {
					piiFields = append(append([]string{}, piiFields...), DetectPIIFields(payloadMap)...)
				}
				redactedPayload, _ := RedactPII(payloadMap, piiFields).(map[string]any)
				if errs, validationErr := ValidateEventAgainstSchema(redactedPayload, &schema); validationErr != nil || len(errs) > 0 {
					rejected++
					continue
				}
			}
		}
		if s.QuotaLimiter != nil && !s.QuotaLimiter.Allow(ev.TenantID, int64(len(payload)+len(dims))) {
			rejected++
			continue
		}
		ev.ContentHash = eventContentHash(ev.TenantID, ev.SessionID, ev.EventName, ev.ClientTS, payload, dims)
		if source != "replay" && s.Repo.EventContentExists(ev.ContentHash) {
			rejected++
			continue
		}
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
		if err := s.insertAndUpdateSession(ev, sess); err != nil {
			rejected++
			continue
		}
		accepted++
		if s.Counters != nil {
			s.Counters.Increment(ev.TenantID)
		}
		if s.Sessions != nil {
			s.Sessions.Update(sess)
			s.Sessions.AddEvent(sid, ev)
		}
	}
	return accepted, rejected
}

func (s *IngestService) insertAndUpdateSession(ev *model.Event, sess *model.Session) error {
	if s.SetTenantContext == nil {
		if err := s.Repo.InsertEvent(ev); err != nil {
			return err
		}
		if err := UpsertEventBuckets(s.Repo.DB, ev); err != nil {
			return err
		}
		sess.EventCount++
		sess.LastSeen = time.Now()
		return s.Repo.UpdateSession(sess)
	}
	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := s.SetTenantContext(tx, ev.TenantID); err != nil {
		tx.Rollback()
		return err
	}
	txRepo := repository.New(tx)
	if err := txRepo.InsertEvent(ev); err != nil {
		tx.Rollback()
		return err
	}
	if err := UpsertEventBuckets(tx, ev); err != nil {
		tx.Rollback()
		return err
	}
	sess.EventCount++
	sess.LastSeen = time.Now()
	if err := txRepo.UpdateSession(sess); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
