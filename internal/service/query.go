package service

import (
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type QueryService struct{ DB *gorm.DB }

func (q *QueryService) EventsByName(name string, limit int) ([]model.Event, error) {
	var list []model.Event
	if err := q.DB.Where("event_name = ?", name).Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
