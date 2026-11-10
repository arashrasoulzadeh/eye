## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Phase 2 — Storage (TimescaleDB)
- [ ] RLS policies per tenant — n/a for the dev SQLite path (Postgres/Timescale-only feature); not implemented or faked

## Phase 9 — Observability & Ops
- [ ] Docker Compose (dev) + K8s manifests (optional) — not built; CLAUDE.md's dev overrides explicitly rule out Docker for local dev (SQLite only), and no production Postgres/Timescale/Kafka deployment exists yet to write real manifests against, so nothing here would be verifiable rather than guesswork

## Phase 10 — Mobile & Server SDKs
- [ ] Android (Kotlin) SDK port — not attempted; no Kotlin/gradle toolchain in this environment (`kotlinc`/`gradle` not found)

## Phase 14 — Advanced Analytics & Ops
- [x] Real-time counters (gRPC streaming live metrics) — `LiveCounterHub` (internal/service/live_counters.go) tracks a per-tenant running count, bumped by IngestService on every accepted event; the new `LiveCounters` gRPC server-streaming RPC (proto/events.proto, served by QueryStreamServer in server/cmd/server/main.go on :50052) sends an immediate snapshot then pushes one update per subsequent event for that tenant until the client disconnects. Verified with real bufconn gRPC client/server tests (tests/server/live_counters_test.go, tests/server/live_counters_grpc_test.go) plus an IngestService test confirming only accepted events bump the counter.
- [ ] SSO for admin panel (OIDC) — not attempted; no real OIDC identity provider is available in this environment to integrate against and verify

## Workflow rule
- After completing each checkbox: run required tests, commit, push.

## Phase 15 — Extended Testing
- [x] Property-based tests for dedup invariants — randomized `testing/quick` coverage verifies content-hash determinism and separation across tenant, session, event, timestamp, payload, and dims identity fields
- [x] Fuzz testing for proto parsing — native Go fuzz target in `tests/server/proto_fuzz_test.go`; seed corpus and a 3-second fuzz run passed
- [x] Chaos/fault injection (kill DB mid-ingest) — `TestIngestRejectsCleanlyWhenDatabaseFails` closes the SQLite handle during ingestion and verifies a safe rejection
- [x] Contract tests SDK ↔ backend — `web-sdk/src/contract.test.ts` asserts the session/ingest paths, headers, and wire payload against the backend REST contract
- [x] Snapshot tests for admin UI — `admin/src/components/Button.test.tsx` now locks the primary button markup with a Vitest snapshot
- [ ] Mutation testing (go-mutesting)
- [x] Performance regression benchmarks — `BenchmarkTenantRateLimiterAllow` records the rate-limiter baseline (`93.33 ns/op`, `0 allocs/op` on Apple M1 Pro); rerun with `go test ./tests/server -run '^$' -bench BenchmarkTenantRateLimiterAllow -benchmem`
- [x] Security tests (auth bypass, tenant isolation) — auth middleware rejection/allowance and cross-tenant isolation cases pass in `tests/server/handler_test.go`, `tenant_or_admin_auth_test.go`, and tenant-scoped service tests
- [x] Schema validation tests — required fields, primitive types, invalid definitions, and clean payloads are covered in `tests/server/schema_validate_test.go`
- [x] Backfill correctness tests — threshold behavior, idempotency, and reruns after new events are covered in `tests/server/aggregator_backfill_test.go`

## Phase 16 — Advanced Events & Aggregations
- [x] Event dedup by content hash — tenant/session-aware SHA-256 content hashes are stored and enforced on both sync and async client ingestion; replay events remain intentionally additive
- [ ] Event TTL per type
- [x] Event versioning — events persist `EventVersion`, default to version 1 for backwards compatibility, reject invalid versions, and preserve versions through replay
- [ ] Distinct count aggregation (HyperLogLog)
- [ ] Percentile aggregations (p50/p95/p99)
- [ ] Sliding window aggregations
- [x] Session-based aggregations (events per session) — added tenant-scoped `GET /analytics/session-events` with ordered per-session event counts and tests
- [ ] Geolocation-based aggregation
- [ ] Device/platform breakdowns
- [x] Anomaly detection on aggregated series — implemented in `internal/service/anomaly.go` with handler and service coverage
- [ ] Aggregation export to webhooks
- [ ] Materialized views
- [ ] Real-time top-N
- [x] Funnel drop-off aggregation — implemented by `AnalyticsService.Funnel` with ordered-step conversion results and tests
- [x] Retention matrix aggregation — implemented by `AnalyticsService.Retention` with daily/weekly cohort rows and tests

## Phase 17 — A/B Testing
- [ ] Statistical significance reporting
- [ ] Auto-stop on winner

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
