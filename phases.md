## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Production hardening
- [ ] PostgreSQL/TimescaleDB production backend with tenant RLS and time-series migrations — driver selection, Timescale hypertable/index migrations, RLS policy generation, and transaction-local tenant context helper are implemented; request transaction wiring remains before enabling RLS
- [ ] Durable collector transport and persisted ingestion queue
- [ ] Prometheus/Grafana dashboards, structured logs, and alerting
- [ ] Automated database backups and restore-drill verification
- [ ] End-to-end Docker deployment test: tenant → key → session → ingest → query
- [ ] Security hardening: secret manager, CORS allowlist, key expiration/revocation, and OIDC roles — CORS allowlist and optional tenant-key expiration are implemented; secret manager, explicit revocation, and OIDC roles remain
- [ ] Pagination, cursor queries, analytics caching, and scalable aggregation workers

## Workflow rule
- After completing each checkbox: run required tests, commit, push.

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
