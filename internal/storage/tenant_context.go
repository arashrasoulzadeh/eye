package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// SetTenantContext sets the transaction-local tenant used by PostgreSQL RLS
// policies. Call it immediately after Begin and before any tenant-scoped
// query. SET LOCAL is transaction-scoped and cannot leak through the pool.
func SetTenantContext(tx *gorm.DB, tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant id is required")
	}
	return tx.Exec("SELECT set_config('app.tenant_id', ?, true)", tenantID).Error
}
