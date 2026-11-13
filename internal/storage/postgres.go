package storage

import "gorm.io/gorm"

// ApplyPostgresFeatures adds production-only time-series structures after
// GORM has created the portable schema. It is deliberately not called for
// SQLite. RLS is enabled in a later migration, after request-scoped tenant
// context is wired for every query path.
func ApplyPostgresFeatures(db *gorm.DB) error {
	statements := []string{
		"CREATE EXTENSION IF NOT EXISTS timescaledb",
		// Timescale requires every unique index on a hypertable to include the
		// partitioning column. The portable GORM schema creates events_pkey on
		// event_id alone, so replace it with an equivalent time-aware uniqueness
		// constraint before converting the already-populated table.
		"ALTER TABLE events DROP CONSTRAINT IF EXISTS events_pkey",
		// The SQLite-to-Postgres migration copies events before the server starts.
		// migrate_data keeps those existing rows while converting the table.
		"SELECT create_hypertable('events', 'ingest_ts', if_not_exists => TRUE, migrate_data => TRUE)",
		"CREATE UNIQUE INDEX IF NOT EXISTS events_event_id_ingest_ts_key ON events (event_id, ingest_ts)",
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
