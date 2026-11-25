# REST and gRPC route inventory

Extracted from server route registrations. Consult handlers for HTTP methods, request schemas and additional authorization checks.

| Route | Authentication wrapper |
|---|---|
| `/healthz` | See handler |
| `/admin-sessions` | See handler |
| `/tenants` | Admin |
| `/tenants/` | Admin |
| `/tenants/data/` | Admin |
| `/tenant-usage/` | Admin |
| `/schemas` | Admin |
| `/schemas/` | Admin |
| `/sessions` | See handler |
| `/intelligence/users` | Admin |
| `/compliance/erase` | Admin |
| `/events` | See handler |
| `/events/` | See handler |
| `/query` | Admin |
| `/query-explorer` | Admin session |
| `/analytics/funnel` | Admin |
| `/analytics/funnel/pinned` | Admin |
| `/analytics/funnel/suggested` | Admin |
| `/analytics/retention` | Admin |
| `/analytics/views` | Admin |
| `/analytics/session-events` | Admin |
| `/analytics/devices` | Admin |
| `/analytics/top-n` | Admin |
| `/analytics/sliding-window` | Admin |
| `/analytics/distinct-sessions` | Admin |
| `/analytics/percentiles` | Admin |
| `/analytics/geography` | Admin |
| `/analytics/buckets` | Admin |
| `/export/events` | Admin |
| `/cohorts` | Admin |
| `/cohorts/` | Admin |
| `/aggregations` | Admin |
| `/aggregations/` | Admin |
| `/aggregations/export` | Admin |
| `/experiments` | Admin |
| `/experiments/` | Admin |
| `/feature-flags` | Admin |
| `/settings` | Admin |
| `/ingest` | Tenant ingestion |
| `/ingest/stream` | Tenant ingestion |
| `/aggregate-counts` | Admin |
| `/compression/assessment` | Admin session |
| `/compression/prune` | Owner role |
| `/metrics` | Admin |
| `/slo` | Admin |
| `/dashboard-cache` | Admin |
| `/questions` | See handler |
| `/questions/` | See handler |
| `/event-options` | Admin |
| `/live-counters` | Admin |
| `/live-session` | Admin |
| `/live-events` | Admin |
| `/metrics/prometheus` | See handler |
| `/alerts` | Admin |
| `/alerts/` | Admin |
| `/webhooks/dead-letter` | Admin |
| `/replay` | Admin |
| `/audit-log` | Admin |
| `/admin-users` | Admin session |
| `/admin-users/` | Admin session |
| `/admin-users/accept` | See handler |
| `/admin-teams` | Admin session |
| `/admin-teams/` | Admin session |
| `/anomalies` | Admin |

## gRPC

- `CreateSession`
- `Ingest`
- `IngestStream`
- `QueryStream`
- `LiveCounters`
