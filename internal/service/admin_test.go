package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
)

func newTestDB_admin_test(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Schema{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAdminService_GetSchema(t *testing.T) {
	db := newTestDB_admin_test(t)
	repo := repository.New(db)
	svc := &AdminService{Repo: repo, Salt: "test-salt"}

	// Test getting non-existent schema
	_, err := svc.GetSchema("nonexistent")
	if err == nil {
		t.Error("expected error for missing schema")
	}

	// Create a schema and test getting it
	db.Create(&model.Schema{
		ID:         "test-schema",
		TenantID:   "t1",
		Name:       "click",
		Version:    1,
		Definition: `{"type":"object"}`,
	})

	schema, err := svc.GetSchema("test-schema")
	if err != nil {
		t.Fatalf("GetSchema: %v", err)
	}
	if schema.ID != "test-schema" {
		t.Errorf("schema.ID = %s, want test-schema", schema.ID)
	}
}

func TestAdminService_ExpiredAPIKeyIsRejected(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Tenant{}); err != nil {
		t.Fatal(err)
	}
	svc := &AdminService{Repo: repository.New(db), Salt: "salt", APIKeyTTL: time.Hour}
	if _, err := svc.CreateTenant("test", "client-key"); err != nil {
		t.Fatal(err)
	}
	var tenant model.Tenant
	if err := db.First(&tenant).Error; err != nil {
		t.Fatal(err)
	}
	expired := time.Now().Add(-time.Minute)
	if err := db.Model(&tenant).Update("api_key_expires_at", expired).Error; err != nil {
		t.Fatal(err)
	}
	if svc.VerifyAPIKey(tenant.ID, "client-key") {
		t.Fatal("expired API key was accepted")
	}
}

func TestAdminService_RevokedAPIKeyIsRejected(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Tenant{}); err != nil {
		t.Fatal(err)
	}
	svc := &AdminService{Repo: repository.New(db), Salt: "salt"}
	tenant, err := svc.CreateTenant("test", "client-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RevokeKey(tenant.ID); err != nil {
		t.Fatal(err)
	}
	if svc.VerifyAPIKey(tenant.ID, "client-key") {
		t.Fatal("revoked API key was accepted")
	}
}
