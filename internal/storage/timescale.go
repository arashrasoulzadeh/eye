package storage

import (
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

// TimescaleRepo is a GORM-backed Repository implementation. In dev it runs
// against SQLite; in prod the same code runs against a TimescaleDB
// hypertable (see "Dev overrides" at the top of phases.md) - RLS and
// hypertable-specific migrations are Postgres/Timescale-only and are not
// applicable (and not exercised) against the dev SQLite database.
type TimescaleRepo struct{ DB *gorm.DB }

func NewTimescaleRepo(db *gorm.DB) *TimescaleRepo { return &TimescaleRepo{DB: db} }

func (r *TimescaleRepo) Migrate() error {
	return r.DB.AutoMigrate(&model.Tenant{}, &model.Session{}, &model.Event{}, &model.Schema{})
}

func (r *TimescaleRepo) InsertEvents(events []*model.Event) error {
	if len(events) == 0 {
		return nil
	}
	return r.DB.Create(&events).Error
}

func (r *TimescaleRepo) GetEvents(tenantID, sessionID string, limit int) ([]model.Event, error) {
	var events []model.Event
	q := r.DB.Model(&model.Event{})
	if tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if sessionID != "" {
		q = q.Where("session_id = ?", sessionID)
	}
	if limit <= 0 {
		limit = 100
	}
	if err := q.Order("ingest_ts asc").Limit(limit).Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (r *TimescaleRepo) CreateSession(s *model.Session) error { return r.DB.Create(s).Error }

func (r *TimescaleRepo) GetSession(id string) (*model.Session, error) {
	var s model.Session
	if err := r.DB.First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *TimescaleRepo) DeleteExpiredEvents(retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention)
	res := r.DB.Where("ingest_ts < ?", cutoff).Delete(&model.Event{})
	return res.RowsAffected, res.Error
}
