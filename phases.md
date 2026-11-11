## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Production hardening
- [ ] PostgreSQL/TimescaleDB production backend with tenant RLS and time-series migrations — driver selection, Timescale hypertable/index migrations, RLS policy generation, and transaction-local tenant context helper are implemented; request transaction wiring remains before enabling RLS
- [ ] Durable collector transport and persisted ingestion queue — database-backed pending stream events and restart replay are implemented; external broker/collector durability remains
- [ ] Prometheus/Grafana dashboards, structured logs, and alerting
- [ ] Automated database backups and restore-drill verification — scripts and a Kubernetes CronJob template are implemented; production object storage and scheduled restore execution remain
- [ ] End-to-end Docker deployment test: tenant → key → session → ingest → query
- [x] Security hardening: secret manager, CORS allowlist, key expiration/revocation, and OIDC roles — CORS allowlist, expiration, explicit tenant-key revocation, optional OIDC email/group allowlists, and `*_FILE` secret mounts are implemented
- [ ] Pagination, cursor queries, analytics caching, and scalable aggregation workers — cursor pagination and a bounded 5-second query cache are implemented; distributed aggregation remains

## Workflow rule
- After completing each checkbox: run required tests, commit, push.

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
