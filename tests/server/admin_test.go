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
	as := &service.AdminService{Repo: r}
	tn, err := as.CreateTenant("acme", "key123")
	if err != nil || tn.Name != "acme" { t.Fatalf("create failed: %v", err) }
	list, _ := as.ListTenants()
	if len(list) != 1 { t.Fatalf("expected 1 tenant got %d", len(list)) }
}
