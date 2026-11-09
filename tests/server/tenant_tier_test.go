package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
)

func newTierTestRepo(t *testing.T) *repository.Repo {
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

func TestTenantRateLimiterDifferentTiersGetDifferentRates(t *testing.T) {
	l := service.NewTenantRateLimiter()

	// "free" tier (5/sec, burst capacity 10): first 10 calls succeed from a
	// fresh bucket, the 11th should be denied.
	allowed := 0
	for i := 0; i < 11; i++ {
		if l.Allow("free-tenant", "free") {
			allowed++
		}
	}
	if allowed != 10 {
		t.Errorf("free tier: expected 10 allowed out of 11 bursty calls, got %d", allowed)
	}

	// "enterprise" tier (500/sec, burst capacity 1000): the same 11 calls
	// should all succeed since its bucket starts far larger.
	allowed = 0
	for i := 0; i < 11; i++ {
		if l.Allow("ent-tenant", "enterprise") {
			allowed++
		}
	}
	if allowed != 11 {
		t.Errorf("enterprise tier: expected all 11 calls allowed, got %d", allowed)
	}
}

func TestTenantRateLimiterUnknownTierFallsBackToFree(t *testing.T) {
	l := service.NewTenantRateLimiter()
	allowed := 0
	for i := 0; i < 11; i++ {
		if l.Allow("t1", "") {
			allowed++
		}
	}
	if allowed != 10 {
		t.Errorf("empty tier should behave like free (10 allowed of 11), got %d", allowed)
	}
}

func TestTenantRateLimiterTracksTenantsIndependently(t *testing.T) {
	l := service.NewTenantRateLimiter()
	// Exhaust tenant A's free-tier burst.
	for i := 0; i < 10; i++ {
		l.Allow("a", "free")
	}
	if l.Allow("a", "free") {
		t.Error("tenant a should be exhausted")
	}
	// Tenant B is unaffected.
	if !l.Allow("b", "free") {
		t.Error("tenant b should have its own fresh bucket")
	}
}

func TestIngestServiceRejectsOverTenantTierLimit(t *testing.T) {
	repo := newTierTestRepo(t)
	tenant := &model.Tenant{ID: "t1", Name: "t1", Tier: "free"}
	if err := repo.DB.Create(tenant).Error; err != nil {
		t.Fatal(err)
	}
	sess := &model.Session{ID: "s1", TenantID: "t1"}
	if err := repo.CreateSession(sess); err != nil {
		t.Fatal(err)
	}

	svc := &service.IngestService{Repo: repo, TierLimiter: service.NewTenantRateLimiter()}

	events := make([]map[string]any, 15)
	for i := range events {
		events[i] = map[string]any{
			"event_id":   "e" + string(rune('a'+i)),
			"session_id": "s1",
			"event_name": "click",
			"client_ts":  float64(1000 + i),
			"payload":    map[string]any{},
			"dims":       map[string]any{},
		}
	}
	accepted, rejected := svc.Ingest(events)
	if accepted != 10 {
		t.Errorf("expected 10 accepted (free tier burst capacity), got %d", accepted)
	}
	if rejected != 5 {
		t.Errorf("expected 5 rejected, got %d", rejected)
	}
}

func TestIngestServiceEnterpriseTierSustainsMoreThrough(t *testing.T) {
	repo := newTierTestRepo(t)
	tenant := &model.Tenant{ID: "t2", Name: "t2", Tier: "enterprise"}
	if err := repo.DB.Create(tenant).Error; err != nil {
		t.Fatal(err)
	}
	sess := &model.Session{ID: "s2", TenantID: "t2"}
	if err := repo.CreateSession(sess); err != nil {
		t.Fatal(err)
	}

	svc := &service.IngestService{Repo: repo, TierLimiter: service.NewTenantRateLimiter()}

	events := make([]map[string]any, 15)
	for i := range events {
		events[i] = map[string]any{
			"event_id":   "e" + string(rune('a'+i)),
			"session_id": "s2",
			"event_name": "click",
			"client_ts":  float64(1000 + i),
			"payload":    map[string]any{},
			"dims":       map[string]any{},
		}
	}
	accepted, _ := svc.Ingest(events)
	if accepted != 15 {
		t.Errorf("enterprise tier should sustain all 15 events, got %d accepted", accepted)
	}
}

func TestAdminServiceSetTier(t *testing.T) {
	repo := newTierTestRepo(t)
	svc := &service.AdminService{Repo: repo}
	tenant, err := svc.CreateTenant("acme", "key1")
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.SetTier(tenant.ID, "pro"); err != nil {
		t.Fatal(err)
	}
	var got model.Tenant
	repo.DB.First(&got, "id = ?", tenant.ID)
	if got.Tier != "pro" {
		t.Errorf("expected tier pro, got %q", got.Tier)
	}

	if err := svc.SetTier(tenant.ID, "nonsense"); err == nil {
		t.Error("expected error for unknown tier")
	}
	if err := svc.SetTier("missing-id", "pro"); err == nil {
		t.Error("expected error for unknown tenant")
	}
}
