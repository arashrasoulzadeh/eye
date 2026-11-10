---
layout: default
title: Eye Analytics Platform
---

# Eye Analytics Platform

A multi-tenant analytics ingestion and query system with real-time dashboards, session tracking, and SDKs for web and server.

Documentation is deployed automatically through GitHub Pages.

## Quick Links

- [Getting Started](getting-started/README.md)
- [API Reference](api/README.md)
- [Web SDK](sdk/web-sdk.md)
- [Go SDK](sdk/go-sdk.md)
- [Admin Components](components/README.md)
- [Architecture](architecture/README.md)
- [Testing & Benchmarks](testing/README.md)

## Features

- **Real-time Event Ingestion** - REST and gRPC endpoints with batching
- **Session Management** - Automatic creation, tracking, live updates
- **Live Dashboards** - SSE for real-time counters and events
- **Multi-tenant** - Isolated tenants with API keys and rate tiers
- **Admin Panel** - Next.js dashboard for management and analytics
- **Web SDK** - TypeScript with auto-tracking, batching, CDN build
- **Go SDK** - Server-side client with API parity

## Architecture

```
Browser SDK ──▶ API Server (Go) ──▶ SQLite/TimescaleDB
                     │
                     ├─▶ gRPC QueryStream
                     ├─▶ SSE Live Streams
                     └─▶ Admin Panel (Next.js)
```

## Quick Start

```bash
# Start API server (port 8080 + gRPC 50052)
make server

# Start admin panel (port 3000)
make admin

# Open http://localhost:3000, login with admin key
```

## Documentation

| Section | Description |
|---------|-------------|
| [Getting Started](getting-started/README.md) | Installation, running locally, environment setup |
| [API Reference](api/README.md) | Complete REST/gRPC endpoint documentation |
| [Web SDK](sdk/web-sdk.md) | TypeScript client, CDN build, auto-tracking |
| [Go SDK](sdk/go-sdk.md) | Server-side Go client |
| [Components](components/README.md) | Admin panel React component library |
| [Architecture](architecture/README.md) | System design, services, data models |
| [Testing](testing/README.md) | Test structure, coverage, benchmarks |

## License

MIT
