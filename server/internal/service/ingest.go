package service

import (
	"encoding/json"
	"time"

	"github.com/eye/eye/server/internal/model"
	"github.com/eye/eye/server/internal/repository"
)

type IngestService struct{ Repo *repository.Repo }

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