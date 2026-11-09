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

type IngestService struct {
	Repo   *repository.Repo
	ExpSvc ExperimentAssigner
}

func (s *IngestService) Ingest(events []map[string]any) (int, int) {
	accepted, rejected := 0, 0
	for _, e := range events {
		sid, _ := e["session_id"].(string)
		eid, _ := e["event_id"].(string)
		if s.Repo.EventExists(sid, eid) { rejected++; continue }
		payload, _ := json.Marshal(e["payload"])
		dims, _ := json.Marshal(e["dims"])
		ev := &model.Event{
			EventID: eid, SessionID: sid, EventName: e["event_name"].(string),
			ClientTS: int64(e["client_ts"].(float64)), IngestTS: time.Now(),
			Payload: string(payload), Dims: string(dims),
			Source: "client", IsAggregated: false,
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
		if err := s.Repo.InsertEvent(ev); err != nil { rejected++; continue }
		sess.EventCount++
		s.Repo.UpdateSession(sess)
		accepted++
	}
	return accepted, rejected
}