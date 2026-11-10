package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// ReplayIngester is the minimal seam ReplayService needs: anything shaped
// like IngestService.Ingest or StreamIngestPipeline.Ingest works, so replay
// re-runs events through the same validation/tagging/aggregation path a
// normal ingest request would, rather than copying rows directly in the DB.
type ReplayIngester interface {
	Ingest(events []map[string]any) (int, int)
}

// ReplayService re-ingests a tenant's historical events for a time range.
//
// Design: replayed events are never ingested under their original
// event_id (that would just dedup-reject against the row already in the
// table). Instead each gets a fresh, distinct event_id (original id +
// "-replay-" + a random suffix) and Source "replay", so replay is additive
// and auditable — original rows are untouched, and a replayed event is
// always distinguishable from the event it was replayed from. Only rows
// with Source != "replay" are ever selected as replay input, so re-running
// Replay over the same range doesn't compound (it replays the originals
// again, not the replay rows from a previous run).
type ReplayService struct {
	DB     *gorm.DB
	Ingest ReplayIngester
}

func randomReplaySuffix() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Replay re-ingests every event for tenantID with ClientTS in
// [fromTS, toTS] (inclusive) that is not itself a previous replay, and
// returns how many were accepted by the ingest path.
func (rs *ReplayService) Replay(tenantID string, fromTS, toTS int64) (int, error) {
	var events []model.Event
	if err := rs.DB.Where(
		"tenant_id = ? AND client_ts >= ? AND client_ts <= ? AND source != ?",
		tenantID, fromTS, toTS, "replay",
	).Find(&events).Error; err != nil {
		return 0, err
	}

	batch := make([]map[string]any, 0, len(events))
	for _, e := range events {
		var payload, dims any
		_ = json.Unmarshal([]byte(e.Payload), &payload)
		_ = json.Unmarshal([]byte(e.Dims), &dims)
		version := e.EventVersion
		if version < 1 {
			version = 1
		}
		batch = append(batch, map[string]any{
			"event_id":      e.EventID + "-replay-" + randomReplaySuffix(),
			"session_id":    e.SessionID,
			"event_name":    e.EventName,
			"event_version": float64(version),
			"client_ts":     float64(e.ClientTS),
			"payload":       payload,
			"dims":          dims,
			"source":        "replay",
		})
	}
	if len(batch) == 0 {
		return 0, nil
	}
	accepted, _ := rs.Ingest.Ingest(batch)
	return accepted, nil
}
