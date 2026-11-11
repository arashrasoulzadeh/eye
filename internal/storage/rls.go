package storage

import "gorm.io/gorm"

// ApplyPostgresRLS enables tenant isolation for tables that carry tenant_id.
// The application role must not own these tables and must establish
// SetTenantContext inside each transaction. Admin/reporting roles should use
// a separate BYPASSRLS role rather than weakening tenant policies.
func ApplyPostgresRLS(db *gorm.DB) error {
	tables := []string{"sessions", "events", "schemas", "aggregations", "cohorts", "experiments", "alert_rules", "feature_flags", "questions"}
	for _, table := range tables {
		policy := "eye_tenant_isolation_" + table
		statement := `DO $$ BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = current_schema() AND tablename = '` + table + `' AND policyname = '` + policy + `') THEN
  EXECUTE 'ALTER TABLE ` + table + ` ENABLE ROW LEVEL SECURITY';
  EXECUTE 'CREATE POLICY ` + policy + ` ON ` + table + ` USING (tenant_id = current_setting(''app.tenant_id'', true)) WITH CHECK (tenant_id = current_setting(''app.tenant_id'', true))';
END IF;
END $$;`
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
