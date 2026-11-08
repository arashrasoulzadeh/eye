package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/repository"
)

type AdminService struct {
	Repo *repository.Repo
	Salt string
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
	t := &model.Tenant{ID: randomHex(16), Name: name, APIKey: a.hashKey(apiKey)}
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
	return t.APIKey == a.hashKey(apiKey)
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

func (a *AdminService) ListSchemas(tenantID string) ([]model.Schema, error) {
	var list []model.Schema
	if err := a.Repo.DB.Where("tenant_id = ?", tenantID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
