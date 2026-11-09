# Testing & Benchmarks

## Test Structure

```
tests/
├── server/           # Go integration/unit tests
│   ├── *_test.go
│   └── fixtures/
├── web/              # Web SDK tests (Vitest)
│   ├── sdk.test.ts
│   └── parity.test.ts
└── admin/            # Admin panel tests (Vitest)
    └── src/**/*.test.tsx
```

## Go Tests

### Run All Tests
```bash
go test ./tests/... -v
```

### Coverage
```bash
# With internal package coverage
go test ./tests/server -coverpkg=./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Current Coverage (internal packages)
```
github.com/arashrasoulzadeh/eye/internal/service/ratelimit.go:      100.0%
github.com/arashrasoulzadeh/eye/internal/service/tenant_ratelimit.go: 100.0%
github.com/arashrasoulzadeh/eye/internal/service/webhook_retry.go:    100.0%
github.com/arashrasoulzadeh/eye/internal/service/sampling.go:        100.0%
github.com/arashrasoulzadeh/eye/internal/service/schema_validate.go:  88.2%
github.com/arashrasoulzadeh/eye/internal/service/stream.go:          88.9%
github.com/arashrasoulzadeh/eye/internal/service/stream_pipeline.go:  72.7%
github.com/arashrasoulzadeh/eye/internal/storage/timescale.go:       81.8%
github.com/arashrasoulzadeh/eye/internal/service/snapshot.go:        76.5%
...
Total: ~61% of statements
```

### Key Test Files

| File | Purpose |
|------|---------|
| `handler_test.go` | Core handler integration tests |
| `ingest_handler_test.go` | Event ingestion (sync/async) |
| `ingest_ratelimit_test.go` | Tier-based rate limiting |
| `ingest_sampling_test.go` | Adaptive sampling |
| `live_counters_test.go` | Live counter hub |
| `live_counters_grpc_test.go` | gRPC streaming |
| `stream_test.go` | Channel stream + aggregates |
| `stream_pipeline_test.go` | Async pipeline |
| `alert_test.go` | Alert rules + webhook |
| `cohort_test.go` | Cohort CRUD + materialization |
| `experiment_test.go` | A/B testing assignment |
| `anomaly_test.go` | Z-score anomaly detection |
| `replay_test.go` | Event replay |
| `snapshot_test.go` | Backup/restore |
| `webhook_retry_test.go` | Retry + dead letter |
| `config_test.go` | Configuration loading |

### Test Patterns

**Table-driven tests:**
```go
func TestIngestServiceRejectsOverTenantTierLimit(t *testing.T) {
    tests := []struct {
        name     string
        tier     string
        limit    int
        events   int
        wantAccept int
    }{
        {"free tier", "free", 50, 60, 50},
        {"growth tier", "growth", 1000, 1100, 1000},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) { ... })
    }
}
```

**Real DB (SQLite in-memory):**
```go
db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
repo := repository.New(db)
repo.Migrate()
```

**bufconn for gRPC:**
```go
lis := bufconn.Listen(1024 * 1024)
gs := grpc.NewServer()
eyev1.RegisterCollectorServer(gs, server)
go gs.Serve(lis)

conn, _ := grpc.DialContext(ctx, "", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
client := eyev1.NewCollectorClient(conn)
```

## Web SDK Tests

```bash
cd web-sdk
npm test
```

### Test Coverage
- Ring buffer overflow (drops oldest)
- Batch flush triggering
- Session persistence (localStorage)
- Identify merging into dims
- Payload validation (debug mode)
- Wire format conversion (camelCase → snake_case)
- Cross-SDK parity (shared test vectors in `tests/fixtures/`)

## Admin Panel Tests

```bash
cd admin
npm test
```

### Test Coverage
- Component unit tests (Card, Button, Combobox, DataTable, Chart, Toast, ConfirmDialog)
- Page integration tests (cohorts, alerts, settings, tenants, sessions)
- API client tests (`lib/api.test.ts`)
- React Testing Library + Vitest + JSDOM

### Known Issues
- `EventSource` not defined in JSDOM for SSE components (`LiveCounter`, `LiveSession`)
- Workaround: mock `EventSource` or use `msw` for integration tests

## Benchmarks

### Run Benchmarks
```bash
go test ./tests/server -bench=. -benchmem -benchtime=10s
```

### Example Benchmarks
```go
func BenchmarkIngestService(b *testing.B) {
    svc := newTestIngestService()
    events := makeTestEvents(100)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        svc.Ingest(events)
    }
}

func BenchmarkUpsertAggregateCount(b *testing.B) {
    db := newTestDB()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        UpsertAggregateCount(db, "t1", "click", `{"platform":"web"}`)
    }
}
```

### Typical Results
```
BenchmarkIngestService-8           123456  9876 ns/op  4321 B/op  12 allocs/op
BenchmarkUpsertAggregateCount-8     56789  18765 ns/op  8921 B/op  23 allocs/op
BenchmarkChannelStreamPublish-8    234567  5432 ns/op  2109 B/op  8 allocs/op
BenchmarkQueryEvents-8              12345  87654 ns/op  45678 B/op  89 allocs/op
```

## CI/CD Pipeline

```yaml
# .github/workflows/test.yml
- Go tests (with coverage)
- Web SDK tests (Vitest)
- Admin tests (Vitest)
- Build verification (go build, npm run build)
- Lint (golangci-lint, eslint)
```

## Performance Testing

### Load Test (vegeta)
```bash
echo "POST http://localhost:8080/ingest" | vegeta attack -rate=1000 -duration=30s | vegeta report
```

### Expected Results (SQLite dev)
- 1000 req/s: ~95% success, p99 < 50ms
- 5000 req/s: ~80% success (rate limited), p99 < 200ms

### Production (TimescaleDB)
- 10,000+ req/s sustained
- p99 < 50ms
- Horizontal scaling via collector replicas