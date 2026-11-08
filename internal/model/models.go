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

type Aggregation struct {
	ID             string `gorm:"primaryKey"`
	TenantID       string
	Name           string
	Rule           string // JSON
	OutputEvent    string
	Threshold      int
	Window         string
	Enabled        bool
	CreatedAt      int64
	UpdatedAt      int64
}

type Schema struct {
	ID         string `gorm:"primaryKey"`
	TenantID   string `gorm:"index"`
	Name       string `gorm:"index"`
	Version    int
	Definition string // JSON schema
	CreatedAt  int64
}

type Experiment struct {
	ID        string `gorm:"primaryKey"`
	TenantID  string
	Name      string
	Variants  string // JSON array
	Traffic   int
	Enabled   bool
	CreatedAt int64
}
