package repository

import (
	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
)

type Repo struct{ DB *gorm.DB }

func New(db *gorm.DB) *Repo { return &Repo{DB: db} }

func (r *Repo) Migrate() error {
	return r.DB.AutoMigrate(
		&model.Tenant{}, &model.Session{}, &model.Event{}, &model.Schema{},
		&model.Aggregation{}, &model.Experiment{}, &model.ExperimentAssignment{}, &model.Cohort{}, &model.CohortMembership{},
		&model.AggregateCount{}, &model.AlertRule{}, &model.RuntimeSettings{},
		&model.WebhookDeadLetter{}, &model.AuditLog{}, &model.FeatureFlag{}, &model.DashboardCache{},
	)
}

// GetSettings returns the singleton runtime-settings row, or
// gorm.ErrRecordNotFound if it has never been written (callers should fall
// back to internal/config.Config defaults in that case).
func (r *Repo) GetSettings() (*model.RuntimeSettings, error) {
	var s model.RuntimeSettings
	if err := r.DB.First(&s, "id = ?", 1).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// UpsertSettings writes the singleton runtime-settings row (ID is forced
// to 1), creating it if absent. Uses an explicit exists-check + Create/
// Updates rather than a bare Save(), since Save() with a non-zero primary
// key issues a plain UPDATE that silently affects zero rows when the
// singleton row doesn't exist yet (the same pitfall found and fixed for
// AggregateCount's zero-value-key case in Phase 8's UpsertAggregateCount).
func (r *Repo) UpsertSettings(s *model.RuntimeSettings) error {
	s.ID = 1
	var existing model.RuntimeSettings
	err := r.DB.First(&existing, "id = ?", 1).Error
	if err == gorm.ErrRecordNotFound {
		return r.DB.Create(s).Error
	}
	if err != nil {
		return err
	}
	return r.DB.Model(&existing).Updates(map[string]any{
		"retention_hours":       s.RetentionHours,
		"rate_limit_per_second": s.RateLimitPerSecond,
		"updated_at":            s.UpdatedAt,
	}).Error
}

// GetTenant looks up a tenant by ID, used by IngestService to resolve a
// session's tenant's Tier for TenantRateLimiter.
func (r *Repo) GetTenant(id string) (*model.Tenant, error) {
	var t model.Tenant
	if err := r.DB.First(&t, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) CreateSession(s *model.Session) error { return r.DB.Create(s).Error }
func (r *Repo) GetSession(id string) (*model.Session, error) {
	var s model.Session
	if err := r.DB.First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}
func (r *Repo) UpdateSession(s *model.Session) error { return r.DB.Save(s).Error }

func (r *Repo) InsertEvent(e *model.Event) error {
	return r.DB.Create(e).Error
}
func (r *Repo) EventExists(sessionID, eventID string) bool {
	var c int64
	// EventID is the model's primary key, so deduplication must use the same
	// global identity. Checking session_id as well would let a duplicate ID
	// from another session reach the async queue and fail only at consumption.
	r.DB.Model(&model.Event{}).Where("event_id = ?", eventID).Count(&c)
	return c > 0
}

func (r *Repo) EventContentExists(contentHash string) bool {
	if contentHash == "" {
		return false
	}
	var c int64
	r.DB.Model(&model.Event{}).Where("content_hash = ?", contentHash).Count(&c)
	return c > 0
}
