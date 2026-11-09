package service

import (
	"errors"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

// ComplianceService implements GDPR-style erasure: given a session_id — the
// only per-user identifier this schema has (there is no separate "user key"
// on Session/Event) — it deletes every Event belonging to that session and
// the Session row itself.
//
// Deliberately NOT touched: AggregateCount. That table is only kept
// up to date by the Phase 7 async /ingest/stream path's consumer (see
// stream.go's UpsertAggregateCount); events ingested through the plain
// synchronous /ingest never increment it. Decrementing it on erasure would
// therefore silently assume every erased event went through the stream
// path, which isn't true in general and could corrupt the aggregate for
// events it never counted in the first place. AggregateCount is treated as
// a historical, write-once-per-stream-event total that erasure does not
// try to retroactively correct — an explicit, documented choice, not an
// oversight.
type ComplianceService struct{ DB *gorm.DB }

var ErrSessionNotFound = errors.New("session not found")

// EraseSession deletes every event for sessionID and the session row
// itself, atomically. Returns the number of events deleted. Erasing an
// unknown session is an error rather than a silent no-op, so callers can
// tell "already gone" apart from "nothing to erase."
func (c *ComplianceService) EraseSession(sessionID string) (int64, error) {
	var deleted int64
	err := c.DB.Transaction(func(tx *gorm.DB) error {
		var s model.Session
		if err := tx.First(&s, "id = ?", sessionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrSessionNotFound
			}
			return err
		}
		res := tx.Where("session_id = ?", sessionID).Delete(&model.Event{})
		if res.Error != nil {
			return res.Error
		}
		deleted = res.RowsAffected
		if err := tx.Delete(&s).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return deleted, nil
}
