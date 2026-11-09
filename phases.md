## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Phase 2 — Storage (TimescaleDB)
- [ ] RLS policies per tenant — n/a for the dev SQLite path (Postgres/Timescale-only feature); not implemented or faked

## Phase 6 — Segmentation & Query API
- [ ] gRPC streaming: `QueryStream` — not implemented; `/query` is a plain REST (paginated) endpoint, no gRPC streaming RPC added yet

## Phase 7 — Stream Processing & Aggregates
- [x] "Producer" in collector — `service.ChannelStream`, an in-process buffered-channel producer/consumer (see `internal/service/stream.go`), not a real Redpanda/Kafka client: CLAUDE.md's dev overrides rule out Docker/external brokers in dev, so this is the honest substitute fulfilling the same architectural role (decoupling ingest from storage via a real concurrent queue + goroutine worker + backpressure), swappable later for a real `StreamProducer`/`StreamConsumer` impl against an actual broker
- [x] "Consumer" → DB inserts — `StreamIngestPipeline.Consume` (run by the `ChannelStream` worker) inserts the event into the dev SQLite store and bumps the session count; there is no TimescaleDB hypertable in dev to insert into (prod-only, per dev overrides), so this checks off the *consumer-driven insert* half honestly, not a hypertable-specific behavior
- [x] Materialized aggregates (counts by dims) — `AggregateCount` table + `UpsertAggregateCount`/`GetAggregateCount`, incremented by the consumer per `(tenant_id, event_name, dims)` as events flow through; readable via `GET /aggregate-counts`
- [x] TDD: end-to-end ingest→query — `tests/server/stream_test.go` (producer/consumer round trip) and `tests/server/stream_pipeline_test.go` (consumer inserts + aggregates, duplicate/unknown-session rejection, and a full async ingest → materialized-aggregate read-back test)

## Phase 8 — Alerting & Anomaly
- [ ] Threshold rules per tenant
- [ ] Webhook + email dispatcher
- [ ] Adaptive sampling under load
- [ ] ML anomaly detection (basic statistical)

## Phase 9 — Observability & Ops
- [ ] Prometheus metrics: ingest rate, latency, errors, queue depth
- [ ] Grafana dashboards
- [ ] Docker Compose (dev) + K8s manifests (optional)

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
- [ ] Pages:
  - [ ] `/cohorts` — define segment rules — no backend (Phase 6); explicit "not implemented" placeholder, no fake data
  - [ ] `/alerts` — threshold rules + webhook config — no backend (Phase 8); explicit "not implemented" placeholder
  - [ ] `/settings` — retention, sampling, rate limits — read-only honest display only; no update API exists yet, so no editable controls were added
- [ ] TDD: Playwright e2e for login + CRUD flows — verified manually end-to-end in the browser instead; no Playwright suite added

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
- [ ] Statistical significance reporting
- [ ] Auto-stop on winner

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
