# Core API contracts

## Health

GET /healthz is unauthenticated and returns JSON {"status":"ok"}. Other methods return 405.

## Sessions

POST /sessions uses tenant credentials and accepts required tenant_id plus optional user_id and device. Returns the session model with ID. Admin-authenticated GET with session_id returns details; otherwise returns a paginated list. Parameters: tenant_id, from_ts, to_ts, filter, limit, offset, sort, dir, include_user_tags. Time bounds are Unix milliseconds. PUT/PATCH with session_id accepts device, ip and user_agent to update metadata.

## Ingestion

POST /ingest or /ingest/stream uses tenant credentials and an events array. Response contains accepted and rejected counts; rejected includes sampled-out events. HTTP 429 indicates rate limiting.

## Query

GET /query uses admin authentication. Parameters: tenant_id, session_id, event_name, dims_contains, sort, from_ts, to_ts, limit, offset and cursor. Default limit is 100. page=true returns items, total, limit and offset; otherwise an array, with X-Next-Cursor when available.

## SQL

GET /query-explorer?tenant_id=... discovers property keys. POST uses admin-user authentication and the body below. Returns columns, rows, truncated and duration_ms. Body limit is 32768 bytes; unknown fields are rejected.

```json
{"tenant_id":"example","sql":"SELECT COUNT(*) FROM events","from_ts":0,"to_ts":0}
```
