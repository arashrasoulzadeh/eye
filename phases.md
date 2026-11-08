## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Phase 1 — Foundation & Contracts
- [x] Define proto: `CreateSession`, `Ingest`, `IngestStream` in `proto/events.proto`
- [x] Generate Go + TS stubs
- [x] `go.mod`, module path `github.com/eye/eye`
- [x] Types: `Event`, `IngestRequest`, `IngestResponse`, `Session`, `CreateSessionRequest/Response`
- [x] Config loader (env + file): port, db url, api key salt
- [x] Logging (structured, slog) + Prometheus metrics scaffold

## Phase 2 — Storage (TimescaleDB)
- [x] Migration: hypertable `events(tenant_id, session_id, event_id, event_name, client_ts, ingest_ts, payload jsonb, dims jsonb)`
- [x] RLS policies per tenant
- [x] `storage.Repository` interface: `InsertEvents`, `GetEvents`, `CreateSession`, `GetSession`
- [x] `storage.TimescaleRepo` impl + unit tests (mock DB)
- [x] TTL policy + retention job

## Phase 3 — Ingestion (Go collector)
- [x] `ingest.Service` with `Ingest(ctx, req)` returning counts
- [x] Idempotency: in-memory + DB dedup on (session_id, event_id)
- [x] Session validation: `GetSession` → tenant match
- [x] Rate limiter: per-IP + per-tenant token bucket middleware
- [x] gRPC server wiring + health check
- [x] TDD: table tests for dedup, validation, rate limit

## Phase 4 — Admin API
- [x] `admin.Service`: CRUD tenants, keys, schemas
- [x] REST/gRPC endpoints: `CreateTenant`, `RotateKey`, `RegisterSchema`
- [x] Auth: admin API key scope
- [x] TDD: handler tests

## Phase 5 — Web SDK
- [x] `EyeClient` class: constructor, `ensureSession`, `track`, `flush`
- [x] `Session` persistence via localStorage
- [x] Bounded ring buffer (max 1000)
- [x] Flush batching (size/time threshold)
- [x] Unit tests (Jest/Vitest): buffer overflow, session reuse, flush

## Phase 6 — Segmentation & Query API
- [x] Query service: filter by tenant/session/event/dims/time range
- [x] Cohort definitions + materialized membership (stream job)
- [x] gRPC streaming: `QueryStream`
- [x] TDD: filter correctness, pagination

## Phase 7 — Stream Processing & Aggregates
- [x] Redpanda/Kafka producer in collector
- [x] Consumer → TimescaleDB hypertable inserts
- [x] Materialized aggregates (counts by dims)
- [x] TDD: end-to-end ingest→query

## Phase 8 — Alerting & Anomaly
- [x] Threshold rules per tenant
- [x] Webhook + email dispatcher
- [x] Adaptive sampling under load
- [x] ML anomaly detection (basic statistical)

## Phase 9 — Observability & Ops
- [x] Prometheus metrics: ingest rate, latency, errors, queue depth
- [x] Grafana dashboards
- [x] Docker Compose (dev) + K8s manifests (optional)
- [x] CI: GitHub Actions (lint, test, build)

## Phase 10 — Mobile & Server SDKs
- [x] iOS (Swift) SDK port
- [x] Android (Kotlin) SDK port
- [x] Go server SDK
- [x] Parity tests

## Phase 11 — Compliance & DR
- [x] GDPR erasure: delete by session/user key
- [x] Automated S3 snapshots
- [x] Restore drill

## Phase 12 — Admin Panel (Next.js)
- [x] Next.js app in `admin/` with App Router
- [x] Auth: single long API key (env `ADMIN_API_KEY`), stored in httpOnly cookie/session
- [x] Pages:
  - [x] `/login` — API key entry
  - [x] `/` dashboard — ingest rate, events count, active sessions, tenants
  - [x] `/tenants` — list/create/delete tenants, rotate keys
  - [x] `/tenants/[id]` — tenant detail, usage, retention settings
  - [x] `/sessions` — list sessions per tenant, event counts
  - [x] `/events` — paginated event browser with filters (event_name, session, time range)
  - [x] `/schemas` — register/list Protobuf schemas
  - [x] `/cohorts` — define segment rules
  - [x] `/alerts` — threshold rules + webhook config
  - [x] `/settings` — retention, sampling, rate limits
- [x] API routes (Next.js) proxy to Go backend via grpc-gateway (REST)
- [x] Components: DataTable, FilterBar, Chart (recharts), AuthGuard
- [x] TDD: Playwright e2e for login + CRUD flows

## Phase 13 — Aggregators & System Events
- [x] Event fields: `source` (client|system), `is_aggregated` (bool), `aggregation_id` (nullable)
- [x] `aggregations` table: id, tenant_id, name, rule (event_name filter, threshold count, window), output_event_name, enabled, created_at, updated_at
- [x] Admin API: CRUD aggregations; option to reprocess/backfill older events
- [x] Time-window aggregations: cron-based rollups per minute/hour/day; config `window` (1m/1h/1d) in aggregation rule
- [x] System events stored like client events for unified querying
- [x] Editable rules: updating an aggregation can recompute/backfill older events
- [x] TDD: aggregator counting, threshold emission, backfill

## Phase 14 — Advanced Analytics & Ops
- [x] Funnel analysis API (ordered event steps, conversion rates)
- [x] Retention / cohort reports (daily/weekly active, churn)
- [x] Real-time counters (gRPC streaming live metrics)
- [x] Event replay (re-ingest historical range)
- [x] SDK debug mode (verbose logging, local validation)
- [x] Webhook delivery with retries + dead-letter queue
- [x] Data export (CSV/JSON) per tenant
- [x] Schema evolution/versioning + migration tooling
- [x] Multi-tenant rate tiers
- [x] Audit log for admin actions
- [x] SSO for admin panel (OIDC)
- [x] Feature flags per tenant

## Workflow rule
- After completing each checkbox: run required tests, commit, push.

## Phase 15 — Extended Testing
- [x] Property-based tests (gopter/fast-check) for dedup invariants
- [x] Fuzz testing for proto parsing
- [x] Chaos/fault injection (kill DB mid-ingest)
- [x] Contract tests (Pact) SDK ↔ backend
- [x] Snapshot tests for admin UI
- [x] Mutation testing (go-mutesting)
- [x] Performance regression benchmarks
- [x] Security tests (auth bypass, tenant isolation)
- [x] Schema validation tests
- [x] Backfill correctness tests

## Phase 16 — Advanced Events & Aggregations
- [x] Event dedup by content hash
- [x] Event TTL per type
- [x] Event versioning
- [x] Distinct count aggregation (HyperLogLog)
- [x] Percentile aggregations (p50/p95/p99)
- [x] Sliding window aggregations
- [x] Session-based aggregations (events per session)
- [x] Geolocation-based aggregation
- [x] Device/platform breakdowns
- [x] Anomaly detection on aggregated series
- [x] Aggregation export to webhooks
- [x] Materialized views
- [x] Real-time top-N
- [x] Funnel drop-off aggregation
- [x] Retention matrix aggregation

## Phase 17 — A/B Testing
- [x] Experiment definitions (variants, traffic split)
- [x] Variant assignment (sticky per session/user)
- [x] Event tagging with experiment_id + variant
- [x] Statistical significance reporting
- [x] Auto-stop on winner

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
