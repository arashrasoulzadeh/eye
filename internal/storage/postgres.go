package storage

import "gorm.io/gorm"

// ApplyPostgresFeatures adds production-only time-series structures after
// GORM has created the portable schema. It is deliberately not called for
// SQLite. RLS is enabled in a later migration, after request-scoped tenant
// context is wired for every query path.
func ApplyPostgresFeatures(db *gorm.DB) error {
	statements := []string{
		"CREATE EXTENSION IF NOT EXISTS timescaledb",
		"SELECT create_hypertable('events', 'ingest_ts', if_not_exists => TRUE)",
		"CREATE INDEX IF NOT EXISTS idx_events_tenant_ingest ON events (tenant_id, ingest_ts DESC)",
		"CREATE INDEX IF NOT EXISTS idx_events_tenant_name_ingest ON events (tenant_id, event_name, ingest_ts DESC)",
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
