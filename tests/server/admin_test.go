package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
)

func TestAdminCreateTenant(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	r.Migrate()
	as := &service.AdminService{Repo: r, Salt: "test-salt"}
	tn, err := as.CreateTenant("acme", "key123")
	if err != nil || tn.Name != "acme" { t.Fatalf("create failed: %v", err) }
	if tn.ID == "" { t.Fatalf("expected generated tenant id") }
	if tn.APIKey == "key123" { t.Fatalf("api key must not be stored in plaintext") }
	list, _ := as.ListTenants()
	if len(list) != 1 { t.Fatalf("expected 1 tenant got %d", len(list)) }
}

func TestAdminRotateKey(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	r.Migrate()
	as := &service.AdminService{Repo: r, Salt: "test-salt"}
	tn, _ := as.CreateTenant("acme", "key123")
	oldHash := tn.APIKey

	newKey, err := as.RotateKey(tn.ID)
	if err != nil { t.Fatalf("rotate failed: %v", err) }
	if newKey == "" || newKey == "key123" { t.Fatalf("expected a fresh plaintext key, got %q", newKey) }

	list, _ := as.ListTenants()
	if list[0].APIKey == oldHash { t.Fatalf("expected stored key hash to change after rotation") }
	if list[0].APIKey == newKey { t.Fatalf("rotated key must be stored hashed, not plaintext") }

	if !as.VerifyAPIKey(tn.ID, newKey) { t.Fatalf("new key should verify") }
	if as.VerifyAPIKey(tn.ID, "key123") { t.Fatalf("old key must no longer verify") }
}

func TestAdminDeleteTenant(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	r.Migrate()
	as := &service.AdminService{Repo: r, Salt: "test-salt"}
	tn, _ := as.CreateTenant("acme", "key123")

	if err := as.DeleteTenant(tn.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	list, _ := as.ListTenants()
	if len(list) != 0 {
		t.Fatalf("expected 0 tenants after delete, got %d", len(list))
	}
	if err := as.DeleteTenant("nonexistent"); err == nil {
		t.Fatalf("expected error deleting nonexistent tenant")
	}
}

func TestAdminRegisterSchema(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	r.Migrate()
	as := &service.AdminService{Repo: r, Salt: "test-salt"}
	tn, _ := as.CreateTenant("acme", "key123")

	sch, err := as.RegisterSchema(tn.ID, "click_event", `{"type":"object"}`)
	if err != nil { t.Fatalf("register failed: %v", err) }
	if sch.Version != 1 { t.Fatalf("expected first version to be 1, got %d", sch.Version) }

	sch2, err := as.RegisterSchema(tn.ID, "click_event", `{"type":"object","v":2}`)
	if err != nil { t.Fatalf("register v2 failed: %v", err) }
	if sch2.Version != 2 { t.Fatalf("expected second version to be 2, got %d", sch2.Version) }

	list, err := as.ListSchemas(tn.ID)
	if err != nil || len(list) != 2 { t.Fatalf("expected 2 registered schema versions, got %d (%v)", len(list), err) }
}
