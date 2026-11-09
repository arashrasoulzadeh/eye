package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type CohortService struct{ DB *gorm.DB }

func randomCohortID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (c *CohortService) Create(tenantID, name, eventName string, minCount int) (*model.Cohort, error) {
	now := time.Now().Unix()
	cohort := &model.Cohort{
		ID: randomCohortID(), TenantID: tenantID, Name: name,
		EventName: eventName, MinCount: minCount,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := c.DB.Create(cohort).Error; err != nil {
		return nil, err
	}
	return cohort, nil
}

func (c *CohortService) List(tenantID string) ([]model.Cohort, error) {
	var list []model.Cohort
	if err := c.DB.Where("tenant_id = ?", tenantID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Materialize (re)computes which sessions belong to the cohort — those with
// at least MinCount events named EventName, scoped to the cohort's tenant —
// and replaces its stored membership with the result. It returns the
// resulting member count.
func (c *CohortService) Materialize(cohortID string) (int, error) {
	var cohort model.Cohort
	if err := c.DB.First(&cohort, "id = ?", cohortID).Error; err != nil {
		return 0, err
	}

	var sessionIDs []string
	err := c.DB.Model(&model.Event{}).
		Select("session_id").
		Where("tenant_id = ? AND event_name = ?", cohort.TenantID, cohort.EventName).
		Group("session_id").
		Having("COUNT(*) >= ?", cohort.MinCount).
		Pluck("session_id", &sessionIDs).Error
	if err != nil {
		return 0, err
	}

	now := time.Now().Unix()
	return len(sessionIDs), c.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("cohort_id = ?", cohortID).Delete(&model.CohortMembership{}).Error; err != nil {
			return err
		}
		for _, sid := range sessionIDs {
			m := &model.CohortMembership{CohortID: cohortID, SessionID: sid, ComputedAt: now}
			if err := tx.Create(m).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *CohortService) ListMembers(cohortID string) ([]string, error) {
	var ids []string
	err := c.DB.Model(&model.CohortMembership{}).
		Where("cohort_id = ?", cohortID).
		Order("session_id").
		Pluck("session_id", &ids).Error
	return ids, err
}
