package service

import (
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type ExperimentService struct{ DB *gorm.DB }

func (e *ExperimentService) Create(name, tenantID, variants string, traffic int) (*model.Experiment, error) {
	exp := &model.Experiment{Name: name, TenantID: tenantID, Variants: variants, Traffic: traffic, Enabled: true, CreatedAt: time.Now().Unix()}
	if err := e.DB.Create(exp).Error; err != nil { return nil, err }
	return exp, nil
}
