package service

import (
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type QueryService struct{ DB *gorm.DB }

func (q *QueryService) EventsByName(name string, limit int) ([]model.Event, error) {
	var list []model.Event
	if err := q.DB.Where("event_name = ?", name).Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// EventFilter is a conjunction of optional filters for Filter. Zero-value
// fields are treated as "unset" and don't constrain the query.
type EventFilter struct {
	TenantID  string
	SessionID string
	EventName string
	// FromTS/ToTS bound ClientTS (inclusive) when set. nil means unbounded.
	FromTS *int64
	ToTS   *int64
	// DimsContains does a substring match against the raw Dims JSON column.
	// This is a pragmatic stand-in for real JSON querying: it works on
	// SQLite (dev) and Postgres/Timescale (prod) alike without a JSON
	// operator dependency, at the cost of being a text match rather than a
	// structural one.
	DimsContains string
}

// Filter returns events matching all set fields of f, ordered by ClientTS
// ascending, with limit/offset pagination.
func (q *QueryService) Filter(f EventFilter, limit, offset int) ([]model.Event, error) {
	query := q.DB.Model(&model.Event{})
	if f.TenantID != "" {
		query = query.Where("tenant_id = ?", f.TenantID)
	}
	if f.SessionID != "" {
		query = query.Where("session_id = ?", f.SessionID)
	}
	if f.EventName != "" {
		query = query.Where("event_name = ?", f.EventName)
	}
	if f.FromTS != nil {
		query = query.Where("client_ts >= ?", *f.FromTS)
	}
	if f.ToTS != nil {
		query = query.Where("client_ts <= ?", *f.ToTS)
	}
	if f.DimsContains != "" {
		query = query.Where("dims LIKE ?", "%"+f.DimsContains+"%")
	}

	var list []model.Event
	if err := query.Order("client_ts asc").Limit(limit).Offset(offset).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
