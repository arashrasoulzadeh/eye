# Analytics System Plan — "eye"

## Goal
Multiplatform, E2E-encrypted, live, high-throughput (>100k events/s) analytics with segmentation, low/zero memory footprint, API-only (no UI) for now.

## Architecture

### 1. Client SDKs (low memory, streaming)
- Platforms: Web (TS) first, then iOS (Swift), Android (Kotlin), Server (Go/Node). Web-first; reuse core crypto + batching logic ported per platform.
- Batched send with bounded in-memory ring buffer (cap N events or T ms); flush on threshold; drop oldest on overflow (zero-growth).
- Anonymous by default: generate per-install pseudonymous ID (UUID); no IP/UA stored unless opted in.
- E2E encryption: payload encrypted client-side with tenant public key (X25519 + AES-256-GCM or libsodium sealed box); server stores ciphertext only. Segment dimensions (event name, coarse time bucket) remain plaintext for routing/aggregation.

### 2. Ingestion (stateless, fast)
- Edge collectors (Go) behind load balancer; accept gRPC (Protobuf) over HTTP/2; validate schema; write to Redpanda/Kafka (partitioned by tenant).
- No per-event state in collector; backpressure via queue.

### 3. Stream processing
- Flink/Redpanda Connect: server never decrypts. Maintain plaintext metadata aggregates (counts by segment dims) and store encrypted raw events.
- Real-time materialized aggregates in TimescaleDB (streamed inserts).

### 4. Storage
- Dev: SQLite via GORM (no Docker). Prod: TimescaleDB. API via grpc-gateway (REST auto-gen from proto). ORM: GORM.
- Row-level tenant isolation via tenant_id + RLS policies. TTL policies for retention; cold storage to S3.

### 5. Query API (API only)
- gRPC (Protobuf) with server-streaming for live queries: filter by tenant, time range, segment dims, event name; return aggregates or paginated encrypted events (client decrypts).
- Auth: API keys + optional mTLS; tenant isolation enforced at DB RLS + API layer.

### 6. Segmentation
- Cohorts: define segment rules (property filters) via API; materialized cohort membership refreshed on stream.
- Partitioning: physical hypertable chunks by tenant/time for isolation & pruning.
- Event segmentation: property-based filters at query time + pre-aggregated segment metrics.

### 7. Deployment
- Single region; self-hosted via Docker Compose (dev) / Kubernetes (prod); edge collectors can run at edge nodes.

### 8. Live delivery
- gRPC bidirectional streaming for real-time subscriptions; server pushes aggregates/events as they arrive.

### 9. SDK behavior
- Offline buffering configurable per app: when enabled, persist to IndexedDB/SQLite with bounded queue + sync on reconnect; otherwise in-memory ring buffer only.
- Server SDK: Go primary.

### 10. Observability
- Prometheus metrics + Grafana dashboards for collectors, queue, DB, SDK health.

### 11. Auth
- API keys for both ingest and query; scoped per tenant; rotatable.

## Security / Privacy
- Anonymous ingestion: strip IPs, hash optional identifiers with tenant salt.
- No E2E encryption: transport security via TLS only; server stores plaintext events. Reduces payload overhead and enables server-side aggregation/querying. Privacy via anonymous pseudonymous IDs and tenant isolation.
- Rate limiting: per-IP + per-tenant token buckets at edge collector.
- GDPR erasure: key shredding per user (delete user's derived sub-key) renders their events unreadable; aggregates retained.
- Audit logs for key access.

## Schema
- Strict: events must conform to registered Protobuf schemas in a schema registry; unknown event types rejected at ingestion.

## Retention
- Default 90d–1y TTL per tenant (configurable); cold archive to S3 before expiry.

## Crypto & SDK
- Crypto: Web Crypto API (native) for browser; platform-native crypto on mobile; Go crypto for server. No third-party crypto deps.
- Key rotation: annual automatic rotation; old keys retained for decryption of historical data.
- SDK bundle size budget: <5KB gzipped.
- Admin API: full CRUD for tenants, keys, schemas.
- Event payload max ~1KB; best-effort ordering (no strict ordering guarantees).
- Timestamps taken from client clock; server records ingest time separately.
- SDK distribution: CDN script tag (web); npm/Maven/CocoaPods later.

## Advanced features
- Server-side alerting: webhook + email when metrics cross tenant-defined thresholds.
- Adaptive sampling: auto-downsample under load at ingestion.
- High-cardinality dims: hash/bucketed to bound cardinality.
- No billing/metering (internal tool).
- SDK ring buffers bounded; server collectors stateless; TimescaleDB hypertable compression; streaming inserts, no large in-memory batches.

## Implementation details
- Control plane + processors all in Go; monorepo; GitHub Actions CI; Docker Compose for orchestration (K8s optional later).

## Phasing (API only)
1. Ingestion + storage + query API (MVP).
2. SDKs (Web + one mobile).
3. Segmentation + cohorts.
4. Real-time dashboards API / streaming subscriptions.
5. Cold storage + retention.

## Reliability
- Idempotent ingest: client assigns event IDs; server dedups on (tenant, event_id).
- Automated DB snapshots to S3 for backup/DR; single-region.

## Compliance
- GDPR only (no HIPAA/SOC2/PCI). Key shredding + erasure per user.

## Testing
- Unit + integration + load (k6) + chaos/fault injection.

## Project meta
- Name: eye. SemVer versioning. API docs via Protobuf/OpenAPI generated.

## Additional features
- Redis cache for hot aggregates; full-text search across events; ML-based anomaly detection alerts.
