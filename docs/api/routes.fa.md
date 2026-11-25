# فهرست مسیرهای REST و gRPC

این فهرست از ثبت مسیرهای سرور استخراج شده است. روش HTTP و قرارداد درخواست باید در هندلر بررسی شود.

| مسیر | احراز هویت |
|---|---|
| `/healthz` | عمومی |
| `/admin-sessions` | طبق پیاده‌سازی |
| `/tenants` | مدیر |
| `/tenants/` | مدیر |
| `/tenants/data/` | مدیر |
| `/tenant-usage/` | مدیر |
| `/schemas` | مدیر |
| `/schemas/` | مدیر |
| `/sessions` | طبق پیاده‌سازی |
| `/intelligence/users` | مدیر |
| `/compliance/erase` | مدیر |
| `/events` | مدیر |
| `/events/` | مدیر |
| `/query` | مدیر |
| `/query-explorer` | نشست مدیر |
| `/analytics/funnel` | مدیر |
| `/analytics/funnel/pinned` | مدیر |
| `/analytics/funnel/suggested` | مدیر |
| `/analytics/retention` | مدیر |
| `/analytics/views` | مدیر |
| `/analytics/session-events` | مدیر |
| `/analytics/devices` | مدیر |
| `/analytics/top-n` | مدیر |
| `/analytics/sliding-window` | مدیر |
| `/analytics/distinct-sessions` | مدیر |
| `/analytics/percentiles` | مدیر |
| `/analytics/geography` | مدیر |
| `/analytics/buckets` | مدیر |
| `/export/events` | مدیر |
| `/cohorts` | مدیر |
| `/cohorts/` | مدیر |
| `/aggregations` | مدیر |
| `/aggregations/` | مدیر |
| `/aggregations/export` | مدیر |
| `/experiments` | مدیر |
| `/experiments/` | مدیر |
| `/feature-flags` | مدیر |
| `/settings` | مدیر |
| `/ingest` | دریافت مستأجر |
| `/ingest/stream` | دریافت مستأجر |
| `/aggregate-counts` | مدیر |
| `/compression/assessment` | نشست مدیر |
| `/compression/prune` | نقش مالک |
| `/metrics` | مدیر |
| `/slo` | مدیر |
| `/dashboard-cache` | مدیر |
| `/questions` | طبق پیاده‌سازی |
| `/questions/` | طبق پیاده‌سازی |
| `/event-options` | مدیر |
| `/live-counters` | مدیر |
| `/live-session` | مدیر |
| `/live-events` | مدیر |
| `/metrics/prometheus` | طبق پیاده‌سازی |
| `/alerts` | مدیر |
| `/alerts/` | مدیر |
| `/webhooks/dead-letter` | مدیر |
| `/replay` | مدیر |
| `/audit-log` | مدیر |
| `/admin-users` | نشست مدیر |
| `/admin-users/` | نشست مدیر |
| `/admin-users/accept` | طبق پیاده‌سازی |
| `/admin-teams` | نشست مدیر |
| `/admin-teams/` | نشست مدیر |
| `/anomalies` | مدیر |

## gRPC

- `CreateSession`
- `Ingest`
- `IngestStream`
- `QueryStream`
- `LiveCounters`
