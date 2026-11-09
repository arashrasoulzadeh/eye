package service

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

// SettingsRepo is the persistence contract SettingsService depends on —
// satisfied by *repository.Repo.
type SettingsRepo interface {
	GetSettings() (*model.RuntimeSettings, error)
	UpsertSettings(*model.RuntimeSettings) error
}

// SettingsService makes retention and the ingest rate limit genuinely
// DB-backed and runtime-updatable: a PUT /settings call takes effect on
// the retention job's next tick and the live rate limiter's next request,
// not just on the next process restart (unlike the env-file-only values
// in internal/config.Config, which seed the defaults below until an admin
// writes a real settings row).
type SettingsService struct {
	Repo SettingsRepo

	DefaultRetentionHours     int
	DefaultRateLimitPerSecond int
}

// Get returns the current settings, falling back to the config-file
// defaults if no row has ever been written.
func (s *SettingsService) Get() (*model.RuntimeSettings, error) {
	settings, err := s.Repo.GetSettings()
	if err == gorm.ErrRecordNotFound {
		return &model.RuntimeSettings{
			ID:                 1,
			RetentionHours:     s.defaultRetention(),
			RateLimitPerSecond: s.defaultRateLimit(),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *SettingsService) defaultRetention() int {
	if s.DefaultRetentionHours > 0 {
		return s.DefaultRetentionHours
	}
	return 24 * 30
}

func (s *SettingsService) defaultRateLimit() int {
	if s.DefaultRateLimitPerSecond > 0 {
		return s.DefaultRateLimitPerSecond
	}
	return 50
}

// Update validates and persists new settings. Zero/negative fields in
// `partial` are left unchanged (not reset to zero) so a caller can update
// just one field.
func (s *SettingsService) Update(partial model.RuntimeSettings) (*model.RuntimeSettings, error) {
	current, err := s.Get()
	if err != nil {
		return nil, err
	}
	next := *current
	if partial.RetentionHours != 0 {
		if partial.RetentionHours < 0 {
			return nil, fmt.Errorf("retention_hours must be positive, got %d", partial.RetentionHours)
		}
		next.RetentionHours = partial.RetentionHours
	}
	if partial.RateLimitPerSecond != 0 {
		if partial.RateLimitPerSecond < 0 {
			return nil, fmt.Errorf("rate_limit_per_second must be positive, got %d", partial.RateLimitPerSecond)
		}
		next.RateLimitPerSecond = partial.RateLimitPerSecond
	}
	next.ID = 1
	next.UpdatedAt = time.Now().Unix()
	if err := s.Repo.UpsertSettings(&next); err != nil {
		return nil, err
	}
	return &next, nil
}

// CurrentRetention returns the live retention window, for consumers (the
// retention job) that need to re-check it on every tick rather than at
// startup. Any error reading the store falls back to the default rather
// than disabling retention entirely.
func (s *SettingsService) CurrentRetention() time.Duration {
	settings, err := s.Get()
	if err != nil {
		return time.Duration(s.defaultRetention()) * time.Hour
	}
	return time.Duration(settings.RetentionHours) * time.Hour
}

// CurrentRateLimit returns the live ingest rate limit (events/sec), for
// consumers (LiveRateLimiter) that need to re-check it on every request.
func (s *SettingsService) CurrentRateLimit() float64 {
	settings, err := s.Get()
	if err != nil {
		return float64(s.defaultRateLimit())
	}
	return float64(settings.RateLimitPerSecond)
}
