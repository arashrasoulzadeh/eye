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
		return gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	case "postgres", "postgresql", "timescale":
		if strings.TrimSpace(cfg.DatabaseURL) == "" {
			return nil, fmt.Errorf("database_url is required when db_driver=%s", cfg.DBDriver)
		}
		return gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported db_driver %q", cfg.DBDriver)
	}
}
