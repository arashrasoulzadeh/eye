package storage

import (
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/config"
)

func TestOpenDatabaseRejectsPostgresWithoutURL(t *testing.T) {
	_, err := OpenDatabase(&config.Config{DBDriver: "postgres"})
	if err == nil {
		t.Fatal("expected database_url validation error")
	}
}

func TestOpenDatabaseRejectsUnknownDriver(t *testing.T) {
	_, err := OpenDatabase(&config.Config{DBDriver: "redis"})
	if err == nil {
		t.Fatal("expected unsupported driver error")
	}
}

func TestSetTenantContextRequiresTenant(t *testing.T) {
	if err := SetTenantContext(nil, ""); err == nil {
		t.Fatal("expected tenant id validation error")
	}
}
