## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Production hardening
- [ ] PostgreSQL/TimescaleDB production backend with tenant RLS and time-series migrations — driver selection, Timescale migrations, transaction-local context, and non-owner RLS integration coverage are implemented; remaining request read-path wiring is required before production enablement
- [ ] Durable collector transport and persisted ingestion queue — database-backed pending events, restart replay, Redis Streams transport, and runtime broker selection are implemented; collector routing remains
- [x] Prometheus/Grafana dashboards, structured logs, and alerting — metrics, Grafana dashboard, JSON logs, Prometheus rules, and an Alertmanager routing template are implemented
- [x] Automated database backups and restore-drill verification — scripts, optional S3 upload, Kubernetes CronJob, and scheduled GitHub restore-drill workflow are implemented; production credentials still need configuration
- [x] End-to-end Docker deployment test: tenant → key → session → ingest → query — GitHub Actions Compose workflow added for the complete authenticated flow
- [x] Security hardening: secret manager, CORS allowlist, key expiration/revocation, and OIDC roles — CORS allowlist, expiration, explicit tenant-key revocation, optional OIDC email/group allowlists, and `*_FILE` secret mounts are implemented
- [ ] Pagination, cursor queries, analytics caching, and scalable aggregation workers — cursor pagination for `/events` and `/query`, plus a bounded 5-second query cache, are implemented; distributed aggregation remains

## Workflow rule
- After completing each checkbox: run required tests, commit, push.

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
