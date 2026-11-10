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
- [ ] SSO for admin panel (OIDC) — not attempted; no real OIDC identity provider is available in this environment to integrate against and verify

## Workflow rule
- After completing each checkbox: run required tests, commit, push.

## Phase 15 — Extended Testing
- [ ] Mutation testing (go-mutesting)

## Phase 16 — Advanced Events & Aggregations

## Phase 17 — A/B Testing

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
