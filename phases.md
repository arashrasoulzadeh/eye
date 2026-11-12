## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Production hardening
- [x] PostgreSQL/TimescaleDB production backend with tenant RLS and time-series migrations — driver selection, Timescale migrations, transaction-local context, tenant-scoped request read/write paths, non-owner integration coverage, role bootstrap, and explicit startup switch are implemented; production credentials still need configuration
- [x] Durable collector transport and persisted ingestion queue — database-backed pending events, restart replay, Redis Streams transport, runtime selection, Redis deployment manifests, and collector-to-API proxy routing are implemented
- [x] Prometheus/Grafana dashboards, structured logs, and alerting — metrics, Grafana dashboard, JSON logs, Prometheus rules, and an Alertmanager routing template are implemented
- [x] Automated database backups and restore-drill verification — scripts, optional S3 upload, Kubernetes CronJob, and scheduled GitHub restore-drill workflow are implemented; production credentials still need configuration
- [x] End-to-end Docker deployment test: tenant → key → session → ingest → query — GitHub Actions Compose workflow added for the complete authenticated flow
- [x] Security hardening: secret manager, CORS allowlist, key expiration/revocation, and OIDC roles — CORS allowlist, expiration, explicit tenant-key revocation, optional OIDC email/group allowlists, and `*_FILE` secret mounts are implemented
- [x] Pagination, cursor queries, analytics caching, and scalable aggregation workers — cursor pagination, bounded query caching, standalone aggregation worker, and PostgreSQL advisory locking are implemented

## Product expansion roadmap
- [x] Multi-user admin workspace — users, invitations, teams, roles, session management, and migration from the single admin key — durable users/invitations/teams/memberships, revocable sessions, OIDC session integration, and shared-key fallback are implemented and tested
- [x] Usage metering, quotas, and billing foundations — period-based event/storage counters, plan limits, enforcement, usage API, and quota alerts
- [x] Advanced analytics workspace — funnels, conversion rates, retention comparisons, breakdowns, dashboard layouts, and shareable saved views
- [x] Alerting expansion — anomaly, funnel, conversion-rate, and absence rules with email, Slack, and PagerDuty destinations — anomaly z-score evaluation, destination-aware payloads, SMTP email, retries, and dead-letter handling are implemented
- [x] Data governance — PII detection/redaction, schema enforcement modes, tenant exports, deletion workflows, and retention policies — configured and opt-in detection, reject/warn schema paths, export/delete endpoints, and live retention settings are implemented
- [x] Observability and reliability — OpenTelemetry traces, SLO/error-budget dashboards, dead-letter replay UI, and failure notifications — OTLP/HTTP server spans with W3C propagation, SLO/error-budget API/UI, replayable dead letters, and operator notifications are implemented and tested
- [x] SDK ecosystem — React Native, Flutter, Python, and Node SDKs with durable offline queues and diagnostics — all four packages now implement durable offline queues, retry-safe flushing, session persistence, and diagnostics with contract coverage
- [ ] Infrastructure automation — Helm chart, Terraform module, managed dependency options, and one-command installation

## Admin product completeness
- [x] Replay supports relative and absolute time ranges with validation and progress feedback
- [ ] Every admin resource page has server-backed list/filter/sort, create/edit/delete actions, detail views, and contextual help
- [ ] Operations pages expose full CRUD and safe destructive-action confirmation for flags, aggregations, experiments, alerts, and dead letters
- [ ] Admin UX has consistent empty/loading/error states, action menus, copy controls, and API-backed pagination across all resource tables

## Workflow rule
- After completing each checkbox: run required tests, commit, push.

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
