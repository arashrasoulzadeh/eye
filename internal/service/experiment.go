package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type ExperimentService struct{ DB *gorm.DB }

func randomExperimentID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (e *ExperimentService) Create(name, tenantID, variants string, traffic int) (*model.Experiment, error) {
	exp := &model.Experiment{ID: randomExperimentID(), Name: name, TenantID: tenantID, Variants: variants, Traffic: traffic, Enabled: true, CreatedAt: time.Now().Unix()}
	if err := e.DB.Create(exp).Error; err != nil { return nil, err }
	return exp, nil
}
