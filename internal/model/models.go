package model

import "time"

type Tenant struct {
	ID              string `gorm:"primaryKey"`
	Name            string
	APIKey          string
	APIKeyExpiresAt *time.Time
	APIKeyRevokedAt *time.Time
	// Tier picks the tenant's ingest rate limit (see service.TierRates /
	// service.TenantRateLimiter). Empty means "free", the lowest tier, so
	// existing rows created before this field existed behave unchanged.
	Tier string
	// Monthly limits are zero when unlimited. They are enforced by the usage
	// quota service, allowing existing tenants to keep their current behavior.
	MonthlyEventLimit   int64
	MonthlyStorageLimit int64
	CreatedAt           time.Time
}

// FeatureFlag is a per-tenant boolean toggle, looked up by (TenantID, Key).
// Nothing in this codebase gates behavior on one yet — the deliverable is
// the capability to define and check flags per tenant, via
// service.FeatureFlagService.
type FeatureFlag struct {
	ID        string `gorm:"primaryKey"`
	TenantID  string `gorm:"index:idx_flag_tenant_key,unique"`
	Key       string `gorm:"index:idx_flag_tenant_key,unique"`
	Enabled   bool
	CreatedAt int64
	UpdatedAt int64
}

type Session struct {
	ID         string `gorm:"primaryKey"`
	TenantID   string `gorm:"index"`
	StartedAt  time.Time
	LastSeen   time.Time
	EventCount int64
	// UserAgent and IP are captured server-side from the request itself
	// (not trusted from the client body), so every SDK gets this for free
	// without needing to compute or send it.
	UserAgent string
	IP        string
	// DeviceInfo is a JSON blob of whatever structured client-side fields
	// the caller supplied (screen/viewport size, language, timezone, OS,
	// etc.) — only things the server genuinely can't derive from the
	// request itself. Opaque to the schema since its shape legitimately
	// differs per platform (browser vs. a Go server process).
	DeviceInfo string
}

type Event struct {
	EventID       string `gorm:"primaryKey"`
	ContentHash   string `gorm:"index"`
	EventVersion  int
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

type PendingStreamEvent struct {
	EventID   string `gorm:"primaryKey"`
	Payload   string
	CreatedAt time.Time
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

// AlertRule is a per-tenant threshold rule: when the materialized,
// cumulative count for (TenantID, EventName, DimsKey) — see AggregateCount —
// reaches Threshold, the evaluation job (see service.RunAlertJob) fires
// WebhookURL. DimsKey empty means "any dims" and is matched against the sum
// of all AggregateCount rows for EventName regardless of dims. There is no
// time-windowing here: AggregateCount is a running lifetime total, not a
// per-window one, so "Window" is deliberately not a field on this rule.
type AlertRule struct {
	ID             string `gorm:"primaryKey"`
	TenantID       string `gorm:"index"`
	EventName      string
	DimsKey        string
	Metric         string // currently only "count" is supported
	Threshold      int64
	WindowSeconds  int64 // used by absence rules; zero means the service default
	WebhookURL     string
	Destination    string // webhook, slack, pagerduty, or email (HTTP gateway)
	Enabled        bool
	LastFiredAt    int64 // 0 if never fired; used to avoid re-firing every tick while still above threshold
	LastFiredCount int64 // the count value at the time of the last fire, so a rule only re-fires once the count has grown
	CreatedAt      int64
	UpdatedAt      int64
}

// RuntimeSettings is a singleton row (ID is always 1) holding
// admin-editable, live-updatable backend settings: retention and the
// ingest rate limit. Unlike internal/config.Config (env-file values read
// once at process startup), a change here takes effect immediately for
// the retention job (next tick) and the ingest rate limiter (next
// request) — see service.SettingsService.
type RuntimeSettings struct {
	ID                 uint `gorm:"primaryKey"`
	RetentionHours     int
	RateLimitPerSecond int
	UpdatedAt          int64
}

// WebhookDeadLetter is a webhook delivery that exhausted its retries without
// a successful response, persisted instead of being silently dropped (see
// service.DispatchWithRetry / RunAlertJob).
type WebhookDeadLetter struct {
	ID          string `gorm:"primaryKey"`
	AlertRuleID string `gorm:"index"`
	URL         string
	Payload     string // JSON
	LastError   string
	Attempts    int
	FailedAt    int64
}

// AuditLog is one admin-auth-gated mutating request, written by the audit
// middleware (see handler.AuditLog) after the request completes. This
// project has a single shared admin API key with no per-user identity, so
// ActorKey is always the fixed label "admin" — there is no way to
// distinguish which human used the key.
type AuditLog struct {
	ID          string `gorm:"primaryKey"`
	Action      string // HTTP method, e.g. "POST"
	Resource    string // request path, e.g. "/tenants/abc123"
	ActorKey    string // always "admin" — see doc comment above
	StatusCode  int
	BodySnippet string // first ~256 bytes of the request body, best-effort
	CreatedAt   int64
}

// AdminUser is the durable identity/role record used by the multi-user admin workspace.
// Authentication remains delegated to the configured OIDC/admin gateway; this table
// stores authorization and lifecycle state without persisting passwords or plaintext invitation tokens.
type AdminUser struct {
	ID              string `gorm:"primaryKey" json:"id"`
	Email           string `gorm:"uniqueIndex" json:"email"`
	Name            string `json:"name"`
	Role            string `json:"role"`   // owner, admin, analyst, viewer
	Status          string `json:"status"` // active, invited, suspended
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
	InviteTokenHash string `json:"-"`
	InviteExpiresAt int64  `json:"invite_expires_at,omitempty"`
}

// DashboardCache holds pre-computed dashboard metrics, refreshed by a
// background job on a configurable interval. The dashboard UI reads from
// this table instead of querying the database directly.
type DashboardCache struct {
	ID             uint   `gorm:"primaryKey"` // always 1
	Events         int64  `json:"events"`
	Sessions       int64  `json:"sessions"`
	Tenants        int64  `json:"tenants"`
	EventsByHour   string `json:"events_by_hour"`   // JSON: [{"label":"00:00","value":10},...]
	EventsByTenant string `json:"events_by_tenant"` // JSON: [{"label":"acme","value":100},...]
	UpdatedAt      int64  `json:"updated_at"`
}

// Question is a saved, tenant-scoped event-count graph definition.
type Question struct {
	ID           string `gorm:"primaryKey"`
	TenantID     string `gorm:"index"`
	Name         string
	EventName    string
	EventNames   string // comma-separated event names
	MatchMode    string // any or all
	ChartType    string
	RangeMode    string // relative or absolute
	RelativeDays int
	FromTS       int64
	ToTS         int64
	CreatedAt    int64
	UpdatedAt    int64
}

// AnalyticsView stores a shareable analytics query configuration.
type AnalyticsView struct {
	ID        string `gorm:"primaryKey" json:"id"`
	TenantID  string `gorm:"index" json:"tenant_id"`
	Name      string `json:"name"`
	Query     string `json:"query"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
