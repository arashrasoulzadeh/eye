# Eye Analytics Platform

[![Go Tests](https://github.com/arashrasoulzadeh/eye/actions/workflows/ci.yml/badge.svg?event=push)](https://github.com/arashrasoulzadeh/eye/actions/workflows/ci.yml)
[![Documentation](https://github.com/arashrasoulzadeh/eye/actions/workflows/docs.yml/badge.svg)](https://arashrasoulzadeh.github.io/eye/)

A multi-tenant analytics ingestion and query system with real-time dashboards, session tracking, and SDKs for web and server.

## Features

- **Real-time Event Ingestion** — REST and gRPC endpoints with batching, deduplication, and schema validation
- **Session Management** — Automatic session creation, tracking, and live updates via SSE
- **Live Dashboards** — Server-Sent Events for real-time counters, session events, and metrics
- **Multi-tenant Architecture** — Isolated tenants with API keys, rate limiting tiers (free → enterprise)
- **Admin Panel** — Next.js dashboard for tenant management, event browsing, cohorts, alerts, experiments
- **Web SDK** — TypeScript client with auto-tracking, session persistence, ring buffer, CDN build
- **Go SDK** — Server-side client with API parity to web SDK
- **Background Jobs** — Retention, cohort materialization, aggregations, alert evaluation, anomaly detection, snapshots
- **Compliance** — GDPR/CCPA user data erasure endpoint

## Quick Start

```bash
# Prerequisites: Go 1.23+, Node.js 20+
git clone https://github.com/arashrasoulzadeh/eye
cd eye

# Start API server (port 8080 REST + 50052 gRPC)
make server

# Start admin panel (port 3000)
make admin

# Open http://localhost:3000
# Login with admin key: dev-admin-key-1234567890
```

## Architecture

```
┌─────────────┐     ┌─────────────────┐     ┌─────────────┐
│  Web SDK    │────▶│  API Server     │────▶│  SQLite/    │
│  (Browser)  │     │  (Go + gRPC)    │     │  TimescaleDB│
└─────────────┘     └────────┬────────┘     └─────────────┘
                             │
                    ┌────────┴────────┐
                    │  Admin Panel    │
                    │  (Next.js)      │
                    └─────────────────┘
```

## Documentation

| Section | Description |
|---------|-------------|
| [Getting Started](docs/getting-started/README.md) | Installation, running locally, environment |
| [API Reference](docs/api/README.md) | Complete REST + gRPC endpoint docs |
| [Web SDK](docs/sdk/web-sdk.md) | TypeScript client, CDN build, auto-track |
| [Go SDK](docs/sdk/go-sdk.md) | Server-side Go client |
| [Components](docs/components/README.md) | Admin panel React components |
| [Architecture](docs/architecture/README.md) | System design, services, data models |
| [Testing](docs/testing/README.md) | Test structure, coverage, benchmarks |

**Full docs:** https://arashrasoulzadeh.github.io/eye/

## API Endpoints

### Sessions
- `POST /sessions` — Create session (tenant or admin key)
- `GET /sessions` — List sessions (admin)
- `PUT /sessions?session_id=xxx` — Update device/IP/UA

### Ingestion
- `POST /ingest` — Batch sync ingestion
- `POST /ingest/stream` — Async ingestion

### Queries
- `POST /query` — Filtered event query
- gRPC `QueryStream` — Streaming query (port 50052)

### Live Streams (SSE, admin auth)
- `GET /live-counters?tenant_id=xxx` — Real-time event count
- `GET /live-session?session_id=xxx` — Session count + last seen
- `GET /live-events?session_id=xxx` — Session event stream

### Admin
- `GET/POST /tenants` — Tenant CRUD, tier management
- `GET/POST /schemas` — Schema registry
- `GET/POST /cohorts` — Cohort definitions + materialization
- `GET/POST /alerts` — Threshold-based webhook alerts
- `GET /dashboard-cache` — Cached metrics (1min refresh)
- `POST /replay` — Replay historical events

## SDKs

### Web SDK (TypeScript)
```bash
cd web-sdk && npm install && npm run build:cdn
```
```html
<script src="https://cdn.example.com/eye.min.js"></script>
<script>
  Eye.init({ apiKey: '...', tenant: '...', endpoint: '...' });
  Eye.track('click', { button: 'buy' });
  Eye.identify('user-123', { plan: 'pro' });
</script>
```

### Go SDK
```go
client := eyesdk.NewClient(eyesdk.Options{
    Endpoint: "http://localhost:8080",
    APIKey:   "tenant-key",
    TenantID: "my-tenant",
})
client.EnsureSession(ctx)
client.Track("purchase", map[string]any{"amount": 99.99}, map[string]any{"plan": "pro"})
client.Flush(ctx)
```

## Testing

```bash
# Go tests (coverage ~61%)
go test ./tests/... -v

# Coverage report
go test ./tests/server -coverpkg=./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out

# Benchmarks
go test ./tests/server -bench=. -benchmem

# Web SDK tests
cd web-sdk && npm test

# Admin tests
cd admin && npm test

# Go SDK tests
cd sdk-go && go test -v
```

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `retention_hours` | 720 (30 days) | Event retention |
| `rate_limit_per_second` | 50 | Free tier limit |
| Tiers | free:50, starter:200, growth:1000, enterprise:5000 | Per-tenant rate limits |

Configure via `PUT /settings { "retention_hours": 168, "rate_limit_per_second": 200 }`

## Deployment

### Development
```bash
make server   # SQLite, in-process streaming
make admin    # Next.js dev server
```

### Production (Docker + TimescaleDB)
```yaml
services:
  api:
    build: .
    environment:
      - DB_HOST=timescaledb
    ports: ["8080:8080", "50052:50052"]
  timescaledb:
    image: timescale/timescaledb:latest-pg16
```

### Admin Panel (Vercel)
```bash
cd admin && npm run build
# Deploy .vercel/output or out/
```

## License

MIT