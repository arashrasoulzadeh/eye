package model

import "time"

type IntelligenceInference struct {
	TenantID    string `gorm:"primaryKey"`
	Kind        string `gorm:"primaryKey"`
	EntityID    string `gorm:"primaryKey"`
	Version     string
	Tags        string
	Predictions string
	Evidence    string
	ScoredAt    time.Time
}

type IntelligenceInferenceRun struct {
	ID        uint   `gorm:"primaryKey"`
	TenantID  string `gorm:"index"`
	Version   string
	Events    int
	Sessions  int
	Users     int
	Error     string
	CreatedAt time.Time
}

type GlobalIntelligenceRelease struct {
	ID     int `gorm:"primaryKey"`
	Bundle string
}
