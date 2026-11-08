## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Phase 1 — Foundation & Contracts
- [ ] Define proto: `CreateSession`, `Ingest`, `IngestStream` in `proto/events.proto`
- [ ] Generate Go + TS stubs
- [ ] `go.mod`, module path `github.com/eye/eye`
- [ ] Types: `Event`, `IngestRequest`, `IngestResponse`, `Session`, `CreateSessionRequest/Response`
- [ ] Config loader (env + file): port, db url, api key salt
- [ ] Logging (structured, slog) + Prometheus metrics scaffold

## Phase 2 — Storage (TimescaleDB)
- [ ] Migration: hypertable `events(tenant_id, session_id, event_id, event_name, client_ts, ingest_ts, payload jsonb, dims jsonb)`
- [ ] RLS policies per tenant
- [ ] `storage.Repository` interface: `InsertEvents`, `GetEvents`, `CreateSession`, `GetSession`
- [ ] `storage.TimescaleRepo` impl + unit tests (mock DB)
- [ ] TTL policy + retention job

## Phase 3 — Ingestion (Go collector)
- [ ] `ingest.Service` with `Ingest(ctx, req)` returning counts
- [ ] Idempotency: in-memory + DB dedup on (session_id, event_id)
- [ ] Session validation: `GetSession` → tenant match
- [ ] Rate limiter: per-IP + per-tenant token bucket middleware
- [ ] gRPC server wiring + health check
- [ ] TDD: table tests for dedup, validation, rate limit

## Phase 4 — Admin API
- [ ] `admin.Service`: CRUD tenants, keys, schemas
- [ ] REST/gRPC endpoints: `CreateTenant`, `RotateKey`, `RegisterSchema`
- [ ] Auth: admin API key scope
- [ ] TDD: handler tests

## Phase 5 — Web SDK
- [ ] `EyeClient` class: constructor, `ensureSession`, `track`, `flush`
- [ ] `Session` persistence via localStorage
- [ ] Bounded ring buffer (max 1000)
- [ ] Flush batching (size/time threshold)
- [ ] Unit tests (Jest/Vitest): buffer overflow, session reuse, flush

## Phase 6 — Segmentation & Query API
- [ ] Query service: filter by tenant/session/event/dims/time range
- [ ] Cohort definitions + materialized membership (stream job)
- [ ] gRPC streaming: `QueryStream`
- [ ] TDD: filter correctness, pagination

## Phase 7 — Stream Processing & Aggregates
- [ ] Redpanda/Kafka producer in collector
- [ ] Consumer → TimescaleDB hypertable inserts
- [ ] Materialized aggregates (counts by dims)
- [ ] TDD: end-to-end ingest→query

## Phase 8 — Alerting & Anomaly
- [ ] Threshold rules per tenant
- [ ] Webhook + email dispatcher
- [ ] Adaptive sampling under load
- [ ] ML anomaly detection (basic statistical)

## Phase 9 — Observability & Ops
- [ ] Prometheus metrics: ingest rate, latency, errors, queue depth
- [ ] Grafana dashboards
- [ ] Docker Compose (dev) + K8s manifests (optional)
- [ ] CI: GitHub Actions (lint, test, build)

## Phase 10 — Mobile & Server SDKs
- [ ] iOS (Swift) SDK port
- [ ] Android (Kotlin) SDK port
- [ ] Go server SDK
- [ ] Parity tests

## Phase 11 — Compliance & DR
- [ ] GDPR erasure: delete by session/user key
- [ ] Automated S3 snapshots
- [ ] Restore drill

## Phase 12 — Admin Panel (Next.js)
- [ ] Next.js app in `admin/` with App Router
- [ ] Auth: single long API key (env `ADMIN_API_KEY`), stored in httpOnly cookie/session
- [ ] Pages:
  - [ ] `/login` — API key entry
  - [ ] `/` dashboard — ingest rate, events count, active sessions, tenants
  - [ ] `/tenants` — list/create/delete tenants, rotate keys
  - [ ] `/tenants/[id]` — tenant detail, usage, retention settings
  - [ ] `/sessions` — list sessions per tenant, event counts
  - [ ] `/events` — paginated event browser with filters (event_name, session, time range)
  - [ ] `/schemas` — register/list Protobuf schemas
  - [ ] `/cohorts` — define segment rules
  - [ ] `/alerts` — threshold rules + webhook config
  - [ ] `/settings` — retention, sampling, rate limits
- [ ] API routes (Next.js) proxy to Go backend via grpc-gateway (REST)
- [ ] Components: DataTable, FilterBar, Chart (recharts), AuthGuard
- [ ] TDD: Playwright e2e for login + CRUD flows

## Phase 13 — Aggregators & System Events
- [ ] Event fields: `source` (client|system), `is_aggregated` (bool), `aggregation_id` (nullable)
- [ ] `aggregations` table: id, tenant_id, name, rule (event_name filter, threshold count, window), output_event_name, enabled, created_at, updated_at
- [ ] Admin API: CRUD aggregations; option to reprocess/backfill older events
- [ ] Time-window aggregations: cron-based rollups per minute/hour/day; config `window` (1m/1h/1d) in aggregation rule
- [ ] System events stored like client events for unified querying
- [ ] Editable rules: updating an aggregation can recompute/backfill older events
- [ ] TDD: aggregator counting, threshold emission, backfill

## Phase 14 — Advanced Analytics & Ops
- [ ] Funnel analysis API (ordered event steps, conversion rates)
- [ ] Retention / cohort reports (daily/weekly active, churn)
- [ ] Real-time counters (gRPC streaming live metrics)
- [ ] Event replay (re-ingest historical range)
- [ ] SDK debug mode (verbose logging, local validation)
- [ ] Webhook delivery with retries + dead-letter queue
- [ ] Data export (CSV/JSON) per tenant
- [ ] Schema evolution/versioning + migration tooling
- [ ] Multi-tenant rate tiers
- [ ] Audit log for admin actions
- [ ] SSO for admin panel (OIDC)
- [ ] Feature flags per tenant

## Workflow rule
- After completing each checkbox: run required tests, commit, push.

## Phase 15 — Extended Testing
- [ ] Property-based tests (gopter/fast-check) for dedup invariants
- [ ] Fuzz testing for proto parsing
- [ ] Chaos/fault injection (kill DB mid-ingest)
- [ ] Contract tests (Pact) SDK ↔ backend
- [ ] Snapshot tests for admin UI
- [ ] Mutation testing (go-mutesting)
- [ ] Performance regression benchmarks
- [ ] Security tests (auth bypass, tenant isolation)
- [ ] Schema validation tests
- [ ] Backfill correctness tests

## Phase 16 — Advanced Events & Aggregations
- [ ] Event dedup by content hash
- [ ] Event TTL per type
- [ ] Event versioning
- [ ] Distinct count aggregation (HyperLogLog)
- [ ] Percentile aggregations (p50/p95/p99)
- [ ] Sliding window aggregations
- [ ] Session-based aggregations (events per session)
- [ ] Geolocation-based aggregation
- [ ] Device/platform breakdowns
- [ ] Anomaly detection on aggregated series
- [ ] Aggregation export to webhooks
- [ ] Materialized views
- [ ] Real-time top-N
- [ ] Funnel drop-off aggregation
- [ ] Retention matrix aggregation

## Phase 17 — A/B Testing
- [ ] Experiment definitions (variants, traffic split)
- [ ] Variant assignment (sticky per session/user)
- [ ] Event tagging with experiment_id + variant
- [ ] Statistical significance reporting
- [ ] Auto-stop on winner

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
