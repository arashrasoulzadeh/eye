package service

import (
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// UsageSnapshot describes a tenant's event and approximate payload usage for a
// calendar month. Storage is intentionally approximate: it sums the persisted
// JSON column lengths and is stable across SQLite and PostgreSQL.
type UsageSnapshot struct {
	TenantID     string `json:"tenant_id"`
	PeriodStart  int64  `json:"period_start"`
	PeriodEnd    int64  `json:"period_end"`
	EventCount   int64  `json:"event_count"`
	PayloadBytes int64  `json:"payload_bytes"`
	StorageBytes int64  `json:"storage_bytes"`
}

type UsageService struct{ DB *gorm.DB }

func (s *UsageService) Current(tenantID string, now time.Time) (UsageSnapshot, error) {
	start := time.Date(now.UTC().Year(), now.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	var row struct {
		EventCount   int64
		PayloadBytes int64
		DimsBytes    int64
	}
	query := s.DB.Model(&model.Event{}).
		Select("count(*) as event_count, coalesce(sum(length(payload)), 0) as payload_bytes, coalesce(sum(length(dims)), 0) as dims_bytes").
		Where("tenant_id = ? AND ingest_ts >= ? AND ingest_ts < ?", tenantID, start, end).
		Scan(&row)
	if query.Error != nil {
		return UsageSnapshot{}, query.Error
	}
	return UsageSnapshot{TenantID: tenantID, PeriodStart: start.Unix(), PeriodEnd: end.Unix(), EventCount: row.EventCount, PayloadBytes: row.PayloadBytes, StorageBytes: row.PayloadBytes + row.DimsBytes}, nil
}
