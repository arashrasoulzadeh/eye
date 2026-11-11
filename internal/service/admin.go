package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
)

type AdminService struct {
	Repo      *repository.Repo
	Salt      string
	APIKeyTTL time.Duration
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *AdminService) hashKey(apiKey string) string {
	h := sha256.Sum256([]byte(a.Salt + ":" + apiKey))
	return hex.EncodeToString(h[:])
}

func (a *AdminService) CreateTenant(name, apiKey string) (*model.Tenant, error) {
	t := &model.Tenant{ID: randomHex(16), Name: name, APIKey: a.hashKey(apiKey), APIKeyExpiresAt: a.keyExpiry()}
	if err := a.Repo.DB.Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func (a *AdminService) ListTenants() ([]model.Tenant, error) {
	var ts []model.Tenant
	if err := a.Repo.DB.Find(&ts).Error; err != nil {
		return nil, err
	}
	return ts, nil
}

// DeleteTenant permanently removes a tenant by ID.
func (a *AdminService) DeleteTenant(tenantID string) error {
	res := a.Repo.DB.Delete(&model.Tenant{}, "id = ?", tenantID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// RotateKey generates a fresh plaintext API key for the tenant, stores only
// its salted hash, and returns the plaintext key once (it cannot be
// recovered afterward).
func (a *AdminService) RotateKey(tenantID string) (string, error) {
	var t model.Tenant
	if err := a.Repo.DB.First(&t, "id = ?", tenantID).Error; err != nil {
		return "", err
	}
	newKey := randomHex(24)
	t.APIKey = a.hashKey(newKey)
	t.APIKeyExpiresAt = a.keyExpiry()
	t.APIKeyRevokedAt = nil
	if err := a.Repo.DB.Save(&t).Error; err != nil {
		return "", err
	}
	return newKey, nil
}

// VerifyAPIKey reports whether the given plaintext key matches the tenant's
// current stored (hashed) key.
func (a *AdminService) VerifyAPIKey(tenantID, apiKey string) bool {
	var t model.Tenant
	if err := a.Repo.DB.First(&t, "id = ?", tenantID).Error; err != nil {
		return false
	}
	return t.APIKeyRevokedAt == nil && (t.APIKeyExpiresAt == nil || t.APIKeyExpiresAt.After(time.Now())) && t.APIKey == a.hashKey(apiKey)
}

// RevokeKey invalidates the current tenant key without issuing a replacement.
func (a *AdminService) RevokeKey(tenantID string) error {
	now := time.Now()
	res := a.Repo.DB.Model(&model.Tenant{}).Where("id = ?", tenantID).Updates(map[string]any{"api_key_revoked_at": &now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (a *AdminService) keyExpiry() *time.Time {
	if a.APIKeyTTL <= 0 {
		return nil
	}
	expires := time.Now().Add(a.APIKeyTTL)
	return &expires
}

// RegisterSchema registers a new version of a named schema for a tenant.
// Versions increment per (tenant, name); prior versions are kept for
// schema evolution/history.
func (a *AdminService) RegisterSchema(tenantID, name, definition string) (*model.Schema, error) {
	var count int64
	if err := a.Repo.DB.Model(&model.Schema{}).
		Where("tenant_id = ? AND name = ?", tenantID, name).
		Count(&count).Error; err != nil {
		return nil, err
	}
	s := &model.Schema{
		ID:         randomHex(16),
		TenantID:   tenantID,
		Name:       name,
		Version:    int(count) + 1,
		Definition: definition,
	}
	if err := a.Repo.DB.Create(s).Error; err != nil {
		return nil, err
	}
	return s, nil
}

// SetTier updates a tenant's rate tier (see service.TierRates). An unknown
// tier is rejected so a typo can't silently drop a tenant onto whatever
// rateForTier's fallback happens to be.
func (a *AdminService) SetTier(tenantID, tier string) error {
	if _, ok := TierRates[tier]; !ok {
		return fmt.Errorf("unknown tier %q", tier)
	}
	res := a.Repo.DB.Model(&model.Tenant{}).Where("id = ?", tenantID).Update("tier", tier)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (a *AdminService) SetQuotas(tenantID string, eventLimit, storageLimit int64) error {
	if eventLimit < 0 || storageLimit < 0 {
		return fmt.Errorf("quota limits cannot be negative")
	}
	res := a.Repo.DB.Model(&model.Tenant{}).Where("id = ?", tenantID).Updates(map[string]any{
		"monthly_event_limit": eventLimit, "monthly_storage_limit": storageLimit,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (a *AdminService) ListSchemas(tenantID string) ([]model.Schema, error) {
	var list []model.Schema
	if err := a.Repo.DB.Where("tenant_id = ?", tenantID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetSchema looks up a single registered schema version by ID, for
// ValidateEventAgainstSchema.
func (a *AdminService) GetSchema(id string) (*model.Schema, error) {
	var s model.Schema
	if err := a.Repo.DB.First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}
