package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// FeatureFlagService provides per-tenant boolean flags. Nothing in this
// codebase gates behavior on a flag yet; the deliverable is the capability
// to define and check them, via IsEnabled.
type FeatureFlagService struct {
	DB *gorm.DB
}

func randomFlagID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Set creates or updates the (tenantID, key) flag.
func (s *FeatureFlagService) Set(tenantID, key string, enabled bool) (*model.FeatureFlag, error) {
	var existing model.FeatureFlag
	err := s.DB.First(&existing, "tenant_id = ? AND key = ?", tenantID, key).Error
	now := time.Now().Unix()
	if err == gorm.ErrRecordNotFound {
		f := &model.FeatureFlag{
			ID: randomFlagID(), TenantID: tenantID, Key: key,
			Enabled: enabled, CreatedAt: now, UpdatedAt: now,
		}
		if err := s.DB.Create(f).Error; err != nil {
			return nil, err
		}
		return f, nil
	}
	if err != nil {
		return nil, err
	}
	existing.Enabled = enabled
	existing.UpdatedAt = now
	if err := s.DB.Model(&existing).Updates(map[string]any{
		"enabled": enabled, "updated_at": now,
	}).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

// List returns all flags for a tenant.
func (s *FeatureFlagService) List(tenantID string) ([]model.FeatureFlag, error) {
	var list []model.FeatureFlag
	if err := s.DB.Where("tenant_id = ?", tenantID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Delete removes a tenant's flag by key.
func (s *FeatureFlagService) Delete(tenantID, key string) error {
	res := s.DB.Where("tenant_id = ? AND key = ?", tenantID, key).Delete(&model.FeatureFlag{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IsEnabled reports whether the given tenant's flag is enabled. A flag that
// was never defined defaults to false (not an error), so callers can gate
// behavior on it without first checking existence.
func (s *FeatureFlagService) IsEnabled(tenantID, key string) bool {
	var f model.FeatureFlag
	if err := s.DB.First(&f, "tenant_id = ? AND key = ?", tenantID, key).Error; err != nil {
		return false
	}
	return f.Enabled
}
