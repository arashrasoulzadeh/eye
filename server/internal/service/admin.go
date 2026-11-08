package service

import (
	"github.com/eye/eye/server/internal/model"
	"github.com/eye/eye/server/internal/repository"
)

type AdminService struct{ Repo *repository.Repo }

func (a *AdminService) CreateTenant(name, apiKey string) (*model.Tenant, error) {
	t := &model.Tenant{Name: name, APIKey: apiKey}
	// reuse repo? repo has no tenant methods; add via DB
	if err := a.Repo.DB.Create(t).Error; err != nil { return nil, err }
	return t, nil
}
func (a *AdminService) ListTenants() ([]model.Tenant, error) {
	var ts []model.Tenant
	if err := a.Repo.DB.Find(&ts).Error; err != nil { return nil, err }
	return ts, nil
}
