package storage

import (
	"os"
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/config"
	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
)

func TestPostgresProductionMigrations(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	db, err := OpenDatabase(&config.Config{DBDriver: "postgres", DatabaseURL: url})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.New(db).Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPostgresFeatures(db); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRLSRejectsCrossTenantRead(t *testing.T) {
	ownerURL, appURL := os.Getenv("DATABASE_URL"), os.Getenv("RLS_DATABASE_URL")
	if ownerURL == "" || appURL == "" {
		t.Skip("DATABASE_URL and RLS_DATABASE_URL are required")
	}
	owner, err := OpenDatabase(&config.Config{DBDriver: "postgres", DatabaseURL: ownerURL})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.New(owner).Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPostgresFeatures(owner); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPostgresRLS(owner); err != nil {
		t.Fatal(err)
	}
	owner.Create(&model.Session{ID: "rls-a", TenantID: "tenant-a", StartedAt: time.Now(), LastSeen: time.Now()})
	owner.Create(&model.Session{ID: "rls-b", TenantID: "tenant-b", StartedAt: time.Now(), LastSeen: time.Now()})
	app, err := OpenDatabase(&config.Config{DBDriver: "postgres", DatabaseURL: appURL})
	if err != nil {
		t.Fatal(err)
	}
	tx := app.Begin()
	if err := SetTenantContext(tx, "tenant-a"); err != nil {
		t.Fatal(err)
	}
	var visible, hidden model.Session
	if err := tx.First(&visible, "id = ?", "rls-a").Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.First(&hidden, "id = ?", "rls-b").Error; err == nil {
		t.Fatal("cross-tenant session was visible")
	}
	tx.Rollback()
}
