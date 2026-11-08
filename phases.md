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
- [x] Migration: `storage.TimescaleRepo.Migrate` auto-migrates the `events` table (tenant_id, session_id, event_id, event_name, client_ts, ingest_ts, payload, dims) via GORM against dev SQLite
- [ ] RLS policies per tenant — n/a for the dev SQLite path (Postgres/Timescale-only feature); not implemented or faked
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
- [x] CI: GitHub Actions (lint, test, build)

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
- [x] Next.js app in `admin/` with App Router
- [x] Auth: single long API key (env `ADMIN_API_KEY`), stored in httpOnly cookie/session
- [ ] Pages:
  - [x] `/login` — API key entry
  - [x] `/` dashboard — ingest rate, events count, active sessions, tenants
  - [x] `/tenants` — list/create/delete tenants, rotate keys
  - [ ] `/tenants/[id]` — tenant detail, usage, retention settings page — not built; list page covers CRUD, no per-tenant detail route
  - [x] `/sessions` — list sessions per tenant, event counts
  - [x] `/events` — paginated event browser with filters (event_name, tenant_id, session_id); no time-range filter yet
  - [x] `/schemas` — register/list schemas (JSON definitions, not strictly Protobuf) with version history
  - [ ] `/cohorts` — define segment rules — no backend (Phase 6); explicit "not implemented" placeholder, no fake data
  - [ ] `/alerts` — threshold rules + webhook config — no backend (Phase 8); explicit "not implemented" placeholder
  - [ ] `/settings` — retention, sampling, rate limits — read-only honest display only; no update API exists yet, so no editable controls were added
- [x] API routes (Next.js) proxy to Go backend (`/api/proxy/[...path]`, attaches the admin key from the httpOnly cookie)
- [ ] Components: DataTable, FilterBar, Chart (recharts), AuthGuard — DataTable and inline filter bars exist; no Chart (recharts); auth protection done via `src/proxy.ts` (Next's middleware-successor) rather than a dedicated AuthGuard component
- [ ] TDD: Playwright e2e for login + CRUD flows — verified manually end-to-end in the browser instead; no Playwright suite added

## Phase 13 — Aggregators & System Events
- [x] Event fields: `source` (client|system), `is_aggregated` (bool), `aggregation_id` (nullable)
- [x] `aggregations` table: id, tenant_id, name, rule (event_name filter, threshold count, window), output_event_name, enabled, created_at, updated_at
- [ ] Admin API: CRUD aggregations; option to reprocess/backfill older events
- [ ] Time-window aggregations: cron-based rollups per minute/hour/day; config `window` (1m/1h/1d) in aggregation rule
- [x] System events stored like client events for unified querying
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
- [x] Experiment definitions (variants, traffic split)
- [ ] Variant assignment (sticky per session/user)
- [ ] Event tagging with experiment_id + variant
- [ ] Statistical significance reporting
- [ ] Auto-stop on winner

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
