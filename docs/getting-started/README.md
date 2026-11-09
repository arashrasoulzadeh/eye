# Eye Analytics Platform

Eye is a multi-tenant analytics ingestion and query system designed for real-time event tracking, session management, and live dashboards.

## Features

- **Real-time Event Ingestion**: REST and gRPC endpoints for high-throughput event collection
- **Session Management**: Automatic session creation, tracking, and live updates
- **Live Dashboards**: Server-Sent Events (SSE) for real-time counters and session events
- **Multi-tenant Architecture**: Isolated tenants with API keys and rate limiting tiers
- **Admin Panel**: Next.js dashboard for tenant management, event browsing, and analytics
- **Web SDK**: TypeScript client with automatic batching, session persistence, and CDN build
- **Go SDK**: Server-side client with parity to web SDK

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+
- SQLite (dev) or TimescaleDB (prod)

### Running Locally

```bash
# Start API server (port 8080 + gRPC 50052)
make server

# Start admin panel (port 3000)
make admin
```

### Environment Variables

Create `.env` in project root:
```env
port: 8080
db_path: eye.db
admin_key: dev-admin-key-1234567890
```

### Accessing the Admin Panel

1. Open http://localhost:3000
2. Login with admin key: `dev-admin-key-1234567890`

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Web SDK    │────▶│  API Server │────▶│  SQLite/    │
│  (browser)  │     │  (Go/gRPC)  │     │  TimescaleDB│
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                    ┌──────┴──────┐
                    │ Admin Panel │
                    │  (Next.js)  │
                    └─────────────┘
```

### Core Components

- **server/cmd/server** - Main API server with REST + gRPC
- **server/cmd/collector** - Ingestion-only collector
- **internal/service** - Business logic (ingest, query, live counters, sessions)
- **internal/handler** - HTTP/gRPC handlers
- **internal/repository** - Database operations
- **admin/** - Next.js admin dashboard
- **web-sdk/** - TypeScript browser SDK
- **sdk-go/** - Go server SDK

## API Reference

### REST Endpoints

#### Sessions
- `POST /sessions` - Create session (tenant API key or admin key)
- `GET /sessions` - List all sessions (admin)
- `PUT/PATCH /sessions?session_id=xxx` - Update session device info, IP, user agent

#### Ingestion
- `POST /ingest` - Batch event ingestion (sync)
- `POST /ingest/stream` - Async ingestion via channel stream

#### Queries
- `POST /query` - Query events with filters
- gRPC `QueryStream` - Streaming query (port 50052)

#### Live Streams (SSE, admin auth required)
- `GET /live-counters?tenant_id=xxx` - Real-time event count per tenant
- `GET /live-session?session_id=xxx` - Real-time session event count + last seen
- `GET /live-events?session_id=xxx` - Real-time event stream for session

#### Admin
- `GET/POST /tenants` - Tenant CRUD
- `GET/POST /schemas` - Schema registry
- `GET /metrics` - System metrics
- `GET /dashboard-cache` - Cached dashboard metrics (refreshed every minute)
- `GET /events` - Browse events
- `POST /replay` - Replay historical events

### gRPC Services (port 50052)

```protobuf
service Collector {
  rpc CreateSession (CreateSessionRequest) returns (CreateSessionResponse);
  rpc Ingest (IngestRequest) returns (IngestResponse);
  rpc IngestStream (stream IngestRequest) returns (IngestResponse);
  rpc QueryStream (QueryStreamRequest) returns (stream QueryStreamResponse);
  rpc LiveCounters (LiveCountersRequest) returns (stream LiveCountersResponse);
}
```

## SDKs

### Web SDK (TypeScript)

```bash
cd web-sdk && npm install
npm run build:cdn  # outputs dist/eye.min.js
```

```html
<script src="https://cdn.example.com/eye.min.js"></script>
<script>
  Eye.init({
    apiKey: "<tenant-api-key>",
    tenant: "<tenant-id>",
    endpoint: "https://api.example.com"
  });
  
  Eye.track("button_click", { button: "signup" });
  Eye.identify("user123", { plan: "pro" });
</script>
```

**Features:**
- Automatic session management (localStorage)
- Bounded ring buffer (1000 events)
- Batched flush (20 events / 5s)
- Auto-track: page views, clicks, forms, scroll, errors
- Debug mode for development

### Go SDK

```go
import "github.com/eye/eye/sdk-go"

client := eyesdk.NewClient(eyesdk.Options{
    Endpoint: "http://localhost:8080",
    APIKey:   "tenant-api-key",
    TenantID: "my-tenant",
})

client.EnsureSession(context.Background())
client.Track("purchase", map[string]any{"amount": 99.99}, map[string]any{"plan": "pro"})
client.Flush(context.Background())
```

## Admin Panel

The Next.js admin panel provides:

- **Dashboard** - Cached metrics (events, sessions, tenants) with hourly/tenant breakdowns
- **Tenants** - Create, delete, rotate API keys, set rate limit tiers
- **Events** - Browse/query events with filters
- **Sessions** - View session details with live event stream
- **Schemas** - Register JSON schemas for validation
- **Cohorts** - Define and materialize user segments
- **Alerts** - Threshold-based webhook alerts
- **Settings** - Configure retention, rate limits

### Development

```bash
cd admin && npm run dev -- --webpack
```

## Testing

```bash
# Go tests
go test ./tests/... -v

# Coverage
go test ./tests/server -coverpkg=./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out

# Benchmarks
go test ./tests/server -bench=. -benchmem

# Web SDK tests
cd web-sdk && npm test

# Admin tests
cd admin && npm test
```

## Configuration

### Rate Limiting Tiers

| Tier | Requests/sec |
|------|-------------|
| free | 50 |
| starter | 200 |
| growth | 1000 |
| enterprise | 5000 |

Set via admin: `PUT /settings { "rate_limit_per_second": 100 }`

### Retention

Default: 720 hours (30 days). Configurable:
`PUT /settings { "retention_hours": 168 }`

### Feature Flags

Per-tenant boolean toggles:
```
POST /feature-flags { "tenant_id": "t1", "key": "new-ui", "enabled": true }
```

## Deployment

### Production (Docker + TimescaleDB)

```yaml
# docker-compose.yml (example)
services:
  api:
    build: .
    environment:
      - DB_HOST=timescaledb
      - DB_PORT=5432
    ports:
      - "8080:8080"
      - "50052:50052"
  
  timescaledb:
    image: timescale/timescaledb:latest-pg16
    environment:
      - POSTGRES_DB=eye
    volumes:
      - pgdata:/var/lib/postgresql/data
```

### Admin Panel (Vercel/Netlify)

```bash
cd admin && npm run build
# Deploy .vercel/output or out/ directory
```

## License

MIT