## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Phase 2 — Storage (TimescaleDB)
- [ ] RLS policies per tenant — n/a for the dev SQLite path (Postgres/Timescale-only feature); not implemented or faked

## Phase 7 — Stream Processing & Aggregates
- [x] "Producer" in collector — `service.ChannelStream`, an in-process buffered-channel producer/consumer (see `internal/service/stream.go`), not a real Redpanda/Kafka client: CLAUDE.md's dev overrides rule out Docker/external brokers in dev, so this is the honest substitute fulfilling the same architectural role (decoupling ingest from storage via a real concurrent queue + goroutine worker + backpressure), swappable later for a real `StreamProducer`/`StreamConsumer` impl against an actual broker
- [x] "Consumer" → DB inserts — `StreamIngestPipeline.Consume` (run by the `ChannelStream` worker) inserts the event into the dev SQLite store and bumps the session count; there is no TimescaleDB hypertable in dev to insert into (prod-only, per dev overrides), so this checks off the *consumer-driven insert* half honestly, not a hypertable-specific behavior
- [x] Materialized aggregates (counts by dims) — `AggregateCount` table + `UpsertAggregateCount`/`GetAggregateCount`, incremented by the consumer per `(tenant_id, event_name, dims)` as events flow through; readable via `GET /aggregate-counts`
- [x] TDD: end-to-end ingest→query — `tests/server/stream_test.go` (producer/consumer round trip) and `tests/server/stream_pipeline_test.go` (consumer inserts + aggregates, duplicate/unknown-session rejection, and a full async ingest → materialized-aggregate read-back test)

## Phase 8 — Alerting & Anomaly
- [x] Threshold rules per tenant — `AlertRule` model + `AlertService` CRUD (`internal/service/alert.go`), REST at `/alerts` (`AlertsHandler`); threshold is evaluated against the Phase 7 materialized `AggregateCount` running totals (summed across dims when the rule's `DimsKey` is empty, or scoped to one dims bucket otherwise) — not a time-windowed metric, since `AggregateCount` itself is a lifetime total, not per-window
- [x] Webhook dispatcher — `RunAlertJob` polls every enabled rule on a ticker (1m) and POSTs a JSON payload via an injectable `WebhookClient` when a rule's threshold is newly met (tracked by `LastFiredCount` so it only re-fires once the count has grown, not every tick); this is a polling-based check against the materialized read model, not a real-time push, same honest simplification as the Phase 6/7/13 jobs. Email dispatcher not implemented — no SMTP infra in dev, left undone rather than faked
- [ ] Adaptive sampling under load — not attempted
- [ ] ML anomaly detection (basic statistical) — not attempted

## Phase 9 — Observability & Ops
- [x] Prometheus metrics: ingest rate (`ingest_events_total{status}`), latency (`ingest_latency_seconds` histogram), errors (`ingest_errors_total{reason}`), and stream queue depth (`stream_queue_depth` gauge, sampled every 1s from the Phase 7 `ChannelStream`'s buffered-channel length) — all wired into the real `/ingest` and `/ingest/stream` request path (`internal/handler/ingest.go`) and exposed via `promhttp.Handler()` at `/metrics/prometheus` (the pre-existing `/metrics` path is left alone — it serves JSON dashboard stats the admin UI already depends on, not Prometheus exposition, so this is a separate path rather than an overload); tested in `tests/server/metrics_ingest_test.go` and `tests/server/stream_test.go`
- [x] Grafana dashboards — `deploy/grafana/eye-dashboard.json`, a real dashboard JSON with 4 panels (ingest rate by status, latency p50/p95/p99 from the histogram, errors by reason, queue depth gauge) querying exactly the metrics above via a Prometheus datasource; not deployed/screenshotted against a running Grafana, just a valid, matching dashboard definition
- [ ] Docker Compose (dev) + K8s manifests (optional) — not built; CLAUDE.md's dev overrides explicitly rule out Docker for local dev (SQLite only), and no production Postgres/Timescale/Kafka deployment exists yet to write real manifests against, so nothing here would be verifiable rather than guesswork

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
