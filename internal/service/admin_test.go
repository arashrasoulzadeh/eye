package service

import (
	"testing"

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