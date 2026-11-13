package storage

import (
	"strings"
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

func TestSQLiteDSNEnablesWriteConcurrencySafety(t *testing.T) {
	for _, path := range []string{"eye.db", "file:eye?mode=rwc"} {
		dsn := sqliteDSN(path)
		if !strings.Contains(dsn, "_busy_timeout=10000") || !strings.Contains(dsn, "_journal_mode=WAL") || !strings.Contains(dsn, "_foreign_keys=on") {
			t.Fatalf("dsn %q does not include SQLite safety options: %s", path, dsn)
		}
	}
}

func TestSetTenantContextRequiresTenant(t *testing.T) {
	if err := SetTenantContext(nil, ""); err == nil {
		t.Fatal("expected tenant id validation error")
	}
}
