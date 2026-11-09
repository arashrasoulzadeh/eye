package model

import "time"

type Tenant struct {
	ID        string `gorm:"primaryKey"`
	Name      string
	APIKey    string
	CreatedAt time.Time
}

type Session struct {
	ID         string `gorm:"primaryKey"`
	TenantID   string `gorm:"index"`
	StartedAt  time.Time
	LastSeen   time.Time
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
	ExperimentID  string
	Variant       string
}

type Aggregation struct {
	ID          string `gorm:"primaryKey"`
	TenantID    string
	Name        string
	Rule        string // JSON
	OutputEvent string
	Threshold   int
	Window      string
	Enabled     bool
	CreatedAt   int64
	UpdatedAt   int64
}

// AggregateCount is a materialized, incrementally-maintained count of
// events per (tenant, event_name, dims) combination, kept up to date by the
// Phase 7 stream pipeline's consumer as events flow through — a real-time
// read model separate from scanning raw events.
type AggregateCount struct {
	TenantID  string `gorm:"primaryKey"`
	EventName string `gorm:"primaryKey"`
	DimsKey   string `gorm:"primaryKey"`
	Count     int64
	UpdatedAt int64
}

type Schema struct {
	ID         string `gorm:"primaryKey"`
	TenantID   string `gorm:"index"`
	Name       string `gorm:"index"`
	Version    int
	Definition string // JSON schema
	CreatedAt  int64
}

// Cohort defines a rule-based segment: sessions that triggered EventName at
// least MinCount times belong to it. Membership is materialized separately
// (see service.CohortService) rather than computed on every read.
type Cohort struct {
	ID        string `gorm:"primaryKey"`
	TenantID  string `gorm:"index"`
	Name      string
	EventName string
	MinCount  int
	CreatedAt int64
	UpdatedAt int64
}

// CohortMembership is one (cohort, session) pairing produced by the most
// recent materialization run. Materialize() replaces a cohort's rows
// wholesale rather than diffing, so membership always reflects a single
// point-in-time computation.
type CohortMembership struct {
	CohortID   string `gorm:"primaryKey"`
	SessionID  string `gorm:"primaryKey"`
	ComputedAt int64
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

// ExperimentAssignment is the sticky (experiment, session) -> variant pairing
// produced the first time a session is assigned a variant. Looked up again on
// every later call for that session so the assignment never changes.
type ExperimentAssignment struct {
	ExperimentID string `gorm:"primaryKey"`
	SessionID    string `gorm:"primaryKey"`
	Variant      string
	AssignedAt   int64
}
