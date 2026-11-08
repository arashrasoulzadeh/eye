package model

import "time"

type Tenant struct {
	ID        string `gorm:"primaryKey"`
	Name      string
	APIKey    string
	CreatedAt time.Time
}

type Session struct {
	ID        string `gorm:"primaryKey"`
	TenantID  string `gorm:"index"`
	StartedAt time.Time
	LastSeen  time.Time
	EventCount int64
}

type Event struct {
	EventID       string `gorm:"primaryKey"`
	SessionID     string `gorm:"index"`
	TenantID      string `gorm:"index"`
	EventName     string
	ClientTS      int64
	IngestTS      time.Time
	Payload       string // JSON
	Dims          string // JSON
	Source        string
	IsAggregated  bool
	AggregationID string
}
