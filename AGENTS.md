# eye

Multi-tenant analytics ingestion/query system. Full design intent: [PLAN.md](PLAN.md). Task checklist with real done/not-done status: [phases.md](phases.md) — trust it over PLAN.md for current state.

## Dev overrides (differ from PLAN.md)
- Storage: SQLite via GORM for dev, no Docker. TimescaleDB only in prod.
- API: grpc-gateway (`google.golang.org/grpc` + `grpc-ecosystem/grpc-gateway`) generates REST from proto.
- ORM: GORM for all DB access.

## Layout
- `proto/events.proto` — gRPC/protobuf contract.
- `internal/{config,handler,metrics,model,repository,service,storage}` — Go backend.
- `server/cmd/{server,collector}` — entrypoints.
- `web-sdk/src` — TS client SDK (Vitest).
- `admin/src/app` — Next.js admin panel (App Router).
- `tests/{server,web}` — Go and TS tests (not colocated with source).

## Commands
- Run server: `make server` (or `cd server && go run ./cmd/server --env=.env`)
- Run admin: `make admin`
- Go tests: `go test ./tests/...` or `go test ./...`
- Web SDK tests: `cd web-sdk && npm test` (vitest)
- Admin build check: `make admin-build`

## Rules (mandatory, see bottom of phases.md)
- TDD: failing test → implementation → refactor. No merge without passing tests.
- After each phases.md checkbox: run tests, commit, push, then mark `[x]`.
- Never mark a checkbox done unless the code genuinely implements it — `phases.md` has been burned by fake checkmarks before; verify by reading code, not by file existence.
- Localization law: all new user-visible admin strings must go through the shared language/translation layer; persisted Persian selection must be read synchronously so the UI does not intentionally flash English before switching.
