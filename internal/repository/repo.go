package repository

import (
	"github.com/eye/eye/internal/model"
	"gorm.io/gorm"
)

type Repo struct{ DB *gorm.DB }

func New(db *gorm.DB) *Repo { return &Repo{DB: db} }

func (r *Repo) Migrate() error {
	return r.DB.AutoMigrate(
		&model.Tenant{}, &model.Session{}, &model.Event{}, &model.Schema{},
		&model.Aggregation{}, &model.Experiment{}, &model.ExperimentAssignment{}, &model.Cohort{}, &model.CohortMembership{},
		&model.AggregateCount{},
	)
}

func (r *Repo) CreateSession(s *model.Session) error { return r.DB.Create(s).Error }
func (r *Repo) GetSession(id string) (*model.Session, error) {
	var s model.Session
	if err := r.DB.First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}
func (r *Repo) UpdateSession(s *model.Session) error { return r.DB.Save(s).Error }

func (r *Repo) InsertEvent(e *model.Event) error {
	return r.DB.Create(e).Error
}
func (r *Repo) EventExists(sessionID, eventID string) bool {
	var c int64
	r.DB.Model(&model.Event{}).Where("session_id = ? AND event_id = ?", sessionID, eventID).Count(&c)
	return c > 0
}
