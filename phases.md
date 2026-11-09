## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Phase 2 — Storage (TimescaleDB)
- [ ] RLS policies per tenant — n/a for the dev SQLite path (Postgres/Timescale-only feature); not implemented or faked

## Phase 8 — Alerting & Anomaly
- [ ] Adaptive sampling under load — not attempted
- [ ] ML anomaly detection (basic statistical) — not attempted

## Phase 9 — Observability & Ops
- [ ] Docker Compose (dev) + K8s manifests (optional) — not built; CLAUDE.md's dev overrides explicitly rule out Docker for local dev (SQLite only), and no production Postgres/Timescale/Kafka deployment exists yet to write real manifests against, so nothing here would be verifiable rather than guesswork

## Phase 10 — Mobile & Server SDKs
- [ ] Android (Kotlin) SDK port — not attempted; no Kotlin/gradle toolchain in this environment (`kotlinc`/`gradle` not found)

## Phase 11 — Compliance & DR
- [ ] GDPR erasure: delete by session/user key
- [ ] Automated S3 snapshots
- [ ] Restore drill

## Phase 12 — Admin Panel (Next.js)
- [ ] `/settings` — retention, sampling, rate limits — still read-only honest display only; no update API exists yet, so no editable controls were added

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
