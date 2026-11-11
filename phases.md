## Dev overrides
- Storage: SQLite (GORM) for dev; TimescaleDB for prod. No Docker required.
- API: grpc-gateway (google.golang.org/grpc + grpc-ecosystem/grpc-gateway) auto-generates REST from proto.
- ORM: GORM for all DB access.

## Phase 10 — Mobile & Server SDKs
- [x] Android (Kotlin) SDK port — dependency-free session, batching, bounded buffering, persistence, and identify support in `sdk-android/`

## Phase 14 — Advanced Analytics & Ops
- [x] SSO for admin panel (OIDC) — standards-based authorization-code flow with discovery, state/nonce validation, JWKS signature verification, optional email allowlist, and secure admin session cookie

## Workflow rule
- After completing each checkbox: run required tests, commit, push.

## Phase 15 — Extended Testing
- [ ] Mutation testing (go-mutesting) — target and scheduled CI job are present; the upstream tool currently crashes under local Go 1.26 and must be run with a compatible Go/tool version before marking complete

## TDD Rule (MANDATORY)
- Every code change must be test-driven: write failing test first, then implementation, then refactor.
- All tests live under `test/` directories (server/test, web-sdk/test, admin/test).
- No code merged without passing tests.
- After each checkbox: run tests, commit, push, then mark checkbox [x].
