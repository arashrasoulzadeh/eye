package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func newSettingsTestRepo(t *testing.T) *repository.Repo {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	repo := repository.New(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestSettingsServiceGetDefaults(t *testing.T) {
	repo := newSettingsTestRepo(t)
	svc := &service.SettingsService{Repo: repo, DefaultRetentionHours: 72, DefaultRateLimitPerSecond: 25}

	s, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if s.RetentionHours != 72 {
		t.Errorf("expected default retention 72, got %d", s.RetentionHours)
	}
	if s.RateLimitPerSecond != 25 {
		t.Errorf("expected default rate limit 25, got %d", s.RateLimitPerSecond)
	}
}

func TestSettingsServiceUpdatePersists(t *testing.T) {
	repo := newSettingsTestRepo(t)
	svc := &service.SettingsService{Repo: repo, DefaultRetentionHours: 72, DefaultRateLimitPerSecond: 25}

	if _, err := svc.Update(model.RuntimeSettings{RetentionHours: 10}); err != nil {
		t.Fatal(err)
	}
	s, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if s.RetentionHours != 10 {
		t.Errorf("expected updated retention 10, got %d", s.RetentionHours)
	}
	// Unspecified field must be left unchanged, not reset.
	if s.RateLimitPerSecond != 25 {
		t.Errorf("expected rate limit to remain the default 25, got %d", s.RateLimitPerSecond)
	}

	// A second update (the pre-existing-row path, not the create path).
	if _, err := svc.Update(model.RuntimeSettings{RateLimitPerSecond: 99}); err != nil {
		t.Fatal(err)
	}
	s, err = svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if s.RetentionHours != 10 || s.RateLimitPerSecond != 99 {
		t.Fatalf("expected retention=10 rate=99, got retention=%d rate=%d", s.RetentionHours, s.RateLimitPerSecond)
	}
}

func TestSettingsServiceUpdateRejectsNegative(t *testing.T) {
	repo := newSettingsTestRepo(t)
	svc := &service.SettingsService{Repo: repo}
	if _, err := svc.Update(model.RuntimeSettings{RetentionHours: -5}); err == nil {
		t.Fatal("expected an error for a negative retention_hours")
	}
}

// TestLiveRateLimiterObservesSettingsChange proves the rate limiter actually
// consults live settings on every call, not just at construction — the
// whole point of Phase 12's "runtime-updatable" rate limit.
func TestLiveRateLimiterObservesSettingsChange(t *testing.T) {
	repo := newSettingsTestRepo(t)
	svc := &service.SettingsService{Repo: repo, DefaultRateLimitPerSecond: 1}
	if _, err := svc.Update(model.RuntimeSettings{RateLimitPerSecond: 1}); err != nil {
		t.Fatal(err)
	}
	limiter := &service.LiveRateLimiter{Provider: svc}

	// Burst capacity is 2x rate, so at rate=1 at most 2 calls succeed
	// back-to-back before the bucket is exhausted.
	allowed := 0
	for i := 0; i < 5; i++ {
		if limiter.Allow() {
			allowed++
		}
	}
	if allowed > 2 {
		t.Fatalf("expected at most 2 allowed calls at rate=1, got %d", allowed)
	}

	// Raise the live rate dramatically; the limiter must observe it on the
	// very next call without being reconstructed.
	if _, err := svc.Update(model.RuntimeSettings{RateLimitPerSecond: 10000}); err != nil {
		t.Fatal(err)
	}
	allowedAfter := 0
	for i := 0; i < 50; i++ {
		if limiter.Allow() {
			allowedAfter++
		}
	}
	if allowedAfter < 40 {
		t.Fatalf("expected the raised live rate limit to allow most of 50 calls, got %d", allowedAfter)
	}
}
