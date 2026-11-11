## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Phase 10 — Mobile & Server SDKs
- [x] Android (Kotlin) SDK port — dependency-free session, batching, bounded buffering, persistence, and identify support in `sdk-android/`

## Phase 14 — Advanced Analytics & Ops
- [ ] SSO for admin panel (OIDC) — not attempted; no real OIDC identity provider is available in this environment to integrate against and verify

## Workflow rule
- After completing each checkbox: run required tests, commit, push.

## Phase 15 — Extended Testing
- [ ] Mutation testing (go-mutesting) — add a reproducible local and CI mutation-testing target

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
