package storage

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/config"
)

// OpenDatabase selects SQLite for development and PostgreSQL/TimescaleDB for
// production. Keeping this decision in one place prevents handlers and jobs
// from accidentally opening a second database with different settings.
func OpenDatabase(cfg *config.Config) (*gorm.DB, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.DBDriver)) {
	case "", "sqlite":
		db, err := gorm.Open(sqlite.Open(sqliteDSN(cfg.DBPath)), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		// SQLite permits only one writer. A single pooled connection plus WAL
		// and a busy timeout prevents async intelligence progress updates from
		// racing ingestion/training writes into "database is locked" failures.
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		return db, nil
	case "postgres", "postgresql", "timescale":
		if strings.TrimSpace(cfg.DatabaseURL) == "" {
			return nil, fmt.Errorf("database_url is required when db_driver=%s", cfg.DBDriver)
		}
		return gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported db_driver %q", cfg.DBDriver)
	}
}

func sqliteDSN(path string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + "_busy_timeout=10000&_journal_mode=WAL&_foreign_keys=on"
}
