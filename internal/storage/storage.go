// Package storage defines the persistence contract for the ingestion
// pipeline. In dev it is backed by SQLite (per the repo's "Dev overrides"),
// and the same interface is meant to be backed by TimescaleDB in prod -
// callers never depend on which.
package storage

import (
	"time"

	"github.com/eye/eye/internal/model"
)

// Repository is the storage contract the rest of the system depends on.
type Repository interface {
	Migrate() error

	InsertEvents(events []*model.Event) error
	GetEvents(tenantID, sessionID string, limit int) ([]model.Event, error)

	CreateSession(s *model.Session) error
	GetSession(id string) (*model.Session, error)

	// DeleteExpiredEvents removes events older than the given retention
	// window and reports how many rows were removed.
	DeleteExpiredEvents(retention time.Duration) (int64, error)
}
