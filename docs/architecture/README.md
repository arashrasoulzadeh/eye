# Architecture

System architecture and design decisions for the Eye Analytics Platform.

## Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Eye Analytics Platform                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐              │
│  │  Web SDK     │    │  Go SDK      │    │  Collector   │              │
│  │  (Browser)   │    │  (Server)    │    │  (gRPC)      │              │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘              │
│         │                   │                   │                       │
│         └───────────────────┼───────────────────┘                       │
│                             ▼                                           │
│                    ┌─────────────────┐                                  │
│                    │   API Server    │                                  │
│                    │  (Go + SQLite)  │                                  │
│                    └────────┬────────┘                                  │
│                             │                                           │
│         ┌───────────────────┼───────────────────┐                       │
│         ▼                   ▼                   ▼                       │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                 │
│  │  Ingest     │    │  Query      │    │  Admin      │                 │
│  │  Service    │    │  Service    │    │  Panel      │                 │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘                 │
│         │                  │                  │                         │
│         ▼                  ▼                  ▼                         │
│  ┌─────────────────────────────────────────────────┐                   │
│  │              SQLite / TimescaleDB               │                   │
│  └─────────────────────────────────────────────────┘                   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Core Services

### IngestService (`internal/service/ingest.go`)

Synchronous event ingestion with:
- Schema validation (optional)
- Deduplication (event_id + session_id)
- Experiment assignment
- Tenant rate limiting (tier-based)
- Live counter increment
- Session event count + last seen update

```go
type IngestService struct {
    Repo        *repository.Repo
    ExpSvc      ExperimentAssigner
    TierLimiter TenantLimiter
    Counters    *LiveCounterHub
    Sessions    *LiveSessionHub
}
```

### StreamIngestPipeline (`internal/service/stream_pipeline.go`)

Async ingestion via in-process channel stream (Phase 7):
- Validates/dedups synchronously
- Publishes to `ChannelStream` (256 buffer)
- Background consumer: inserts events, updates aggregates
- Decouples HTTP request from DB write

### LiveCounterHub (`internal/service/live_counters.go`)

Per-tenant running event count with gRPC/SSE streaming:
- `Increment(tenantID)` - called per accepted event
- `Subscribe(tenantID)` - returns channel for SSE/gRPC
- `Snapshot(tenantID)` - current count

### LiveSessionHub (`internal/service/live_session.go`)

Per-session real-time updates:
- `Update(session)` - event count + last seen
- `AddEvent(sessionID, event)` - new event broadcast
- `Subscribe(sessionID)` - session updates (SSE)
- `SubscribeEvents(sessionID)` - event stream (SSE)

### QueryService (`internal/service/query.go`)

Event querying with filters:
- Tenant, session, event name, time range, dims contains
- Pagination via `page_size`
- Used by REST `/query` and gRPC `QueryStream`

### AdminService (`internal/service/admin.go`)

Tenant/schema/cohort/alert/experiment management:
- CRUD operations
- API key rotation
- Tier management

## Data Models

### Session
```go
type Session struct {
    ID         string `gorm:"primaryKey"`
    TenantID   string `gorm:"index"`
    StartedAt  time.Time
    LastSeen   time.Time
    EventCount int64
    UserAgent  string
    IP         string
    DeviceInfo string // JSON
}
```

### Event
```go
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
```

### AggregateCount (Materialized)
```go
type AggregateCount struct {
    TenantID  string `gorm:"primaryKey"`
    EventName string `gorm:"primaryKey"`
    DimsKey   string `gorm:"primaryKey"`
    Count     int64
    UpdatedAt int64
}
```

## Real-time Features

### SSE Endpoints (Admin Auth)
| Endpoint | Query Param | Data Format |
|----------|-------------|-------------|
| `/live-counters` | `tenant_id` | `{"count": 123}` |
| `/live-session` | `session_id` | `{"event_count": 5, "last_seen": "..."}` |
| `/live-events` | `session_id` | `{"events": [...]}` |

### gRPC LiveCounters
```protobuf
rpc LiveCounters (LiveCountersRequest) returns (stream LiveCountersResponse);
```
Serves same `LiveCounterHub` as SSE endpoint.

## Background Jobs

| Job | Interval | Purpose |
|-----|----------|---------|
| Retention | 1 hour | Delete expired events |
| Cohort Materialization | 5 min | Compute cohort membership |
| Aggregation | 5 min | Materialize aggregate counts |
| Alert Evaluation | 1 min | Check thresholds, fire webhooks |
| Anomaly Sampling | 5 min | Z-score anomaly detection |
| Snapshot | 1 hour | Local filesystem DB backup |
| Dashboard Cache | 1 min | Pre-compute dashboard metrics |

All jobs respect context cancellation for graceful shutdown.

## Rate Limiting

### Tenant Tiers
```go
var TierRates = map[string]int{
    "free":       50,
    "starter":    200,
    "growth":     1000,
    "enterprise": 5000,
}
```

- Applied per-tenant in `IngestService`
- Async stream path has separate limiter (not yet wired)
- Configurable via `PUT /settings`

### Adaptive Sampling
```go
type SamplingController struct {
    // Adjusts sample rate based on stream queue depth
    UpdateRate(queueDepth int) float64
}
```
Shared across sync/async paths - backed-up queue throttles both.

## Settings (Live-Updatable)

```go
type RuntimeSettings struct {
    ID                  uint `gorm:"primaryKey"` // always 1
    RetentionHours      int
    RateLimitPerSecond  int
    UpdatedAt           int64
}
```
- Singleton row (ID=1)
- Changes take effect immediately (next job tick / next request)
- Unlike env config which requires restart

## Feature Flags

Per-tenant boolean toggles:
```go
type FeatureFlag struct {
    ID        string `gorm:"primaryKey"`
    TenantID  string `gorm:"index:idx_flag_tenant_key,unique"`
    Key       string `gorm:"index:idx_flag_tenant_key,unique"`
    Enabled   bool
}
```
Checked via `FeatureFlagService.Get(tentantID, key)`.

## Experiments (A/B Testing)

```go
type Experiment struct {
    ID       string
    TenantID string
    Name     string
    Variants []string // JSON array
    Traffic  int      // 0-100
    Enabled  bool
}
```

Sticky assignment:
```go
type ExperimentAssignment struct {
    ExperimentID string `gorm:"primaryKey"`
    SessionID    string `gorm:"primaryKey"`
    Variant      string
    AssignedAt   int64
}
```

## Compliance (GDPR/CCPA)

```go
POST /compliance/erase { "tenant_id": "t1", "user_id": "u1" }
```
- Deletes sessions + events for user
- Idempotent

## Webhook Retries + Dead Letters

```go
DispatchWithRetry(url, payload, maxRetries=3, backoff=2s)
// On exhaustion:
WebhookDeadLetter{ AlertRuleID, URL, Payload, LastError, Attempts, FailedAt }
```
Listed at `/webhooks/dead-letter`.

## Audit Log

All admin mutations logged via middleware:
```go
type AuditLog struct {
    ID           string
    Action       string // HTTP method
    Resource     string // request path
    ActorKey     string // always "admin"
    StatusCode   int
    BodySnippet  string // first 256 bytes
    CreatedAt    int64
}
```

## Development vs Production

| Aspect | Development | Production |
|--------|-------------|------------|
| Database | SQLite (GORM) | TimescaleDB |
| Streaming | ChannelStream (in-process) | Kafka/Redpanda |
| Snapshots | Local filesystem | S3/GCS |
| Rate Limiting | In-memory | Redis/DB-backed |
| Deployment | `go run` | Docker + K8s |

## Security

- Admin key never exposed to browser (tenant keys only)
- `TenantOrAdminAuth` for session creation (public SDK access)
- `AdminAuth` for all other admin endpoints
- Audit middleware inside auth (unauthed requests not logged)
- Single shared admin key (no per-user identity)
- SQL injection prevented via GORM parameterized queries