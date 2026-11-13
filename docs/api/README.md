# API Reference

Complete API documentation for the Eye Analytics Platform.

## Base URLs

- **REST API**: `http://localhost:8080`
- **gRPC API**: `localhost:50052`
- **Admin Panel**: `http://localhost:3000`

## Authentication

| Endpoint | Auth Type |
|----------|-----------|
| `/sessions` (POST) | Tenant API Key OR Admin Key |
| `/ingest`, `/ingest/stream` | Tenant API Key |
| `/query` | Admin Key |
| `/live-*` | Admin Key (via cookie) |
| `/tenants/*`, `/schemas/*`, `/cohorts/*`, etc. | Admin Key |

### Admin Key
Header: `x-admin-key: <admin_key>` (from `.env`)

### Tenant API Key
Header: `x-api-key: <tenant_api_key>`

---

## Sessions

### Create Session
```http
POST /sessions
Content-Type: application/json
x-api-key: <tenant_api_key>

{
  "tenant_id": "my-tenant",
  "device": {
    "screenWidth": 1920,
    "screenHeight": 1080,
    "platform": "MacIntel",
    "timezone": "America/Los_Angeles"
  }
}
```

**Response:**
```json
{
  "ID": "a1b2c3d4e5f6...",
  "TenantID": "my-tenant",
  "StartedAt": "2024-01-15T10:30:00Z",
  "LastSeen": "2024-01-15T10:30:00Z",
  "EventCount": 0,
  "UserAgent": "Mozilla/5.0...",
  "IP": "192.168.1.1",
  "DeviceInfo": "{\"screenWidth\":1920,...}"
}
```

### Update Session
```http
PUT /sessions?session_id=a1b2c3d4e5f6...
Content-Type: application/json
x-admin-key: <admin_key>

{
  "device": { "screenWidth": 2560 },
  "ip": "10.0.0.1",
  "user_agent": "Custom Agent"
}
```

### List Sessions (Admin)
```http
GET /sessions
x-admin-key: <admin_key>
```

---

## Event Ingestion

### Batch Ingest (Sync)
```http
POST /ingest
Content-Type: application/json
x-api-key: <tenant_api_key>

{
  "events": [
    {
      "session_id": "a1b2c3d4e5f6...",
      "event_id": "evt-001",
      "event_name": "button_click",
      "client_ts": 1705315800000,
      "payload": { "button": "signup", "page": "/pricing" },
      "dims": { "plan": "pro", "source": "web" },
      "source": "client"
    }
  ]
}
```

**Response:**
```json
{ "accepted": 1, "rejected": 0 }
```

### Stream Ingest (Async)
```http
POST /ingest/stream
Content-Type: application/json
x-api-key: <tenant_api_key>

{ "events": [...] }
```

Returns immediately, processes asynchronously.

---

## Queries

### Query Events
```http
POST /query
Content-Type: application/json
x-admin-key: <admin_key>

{
  "tenant_id": "my-tenant",
  "session_id": "a1b2c3d4e5f6...",
  "event_name": "button_click",
  "from_ts": 1705276800000,
  "to_ts": 1705363200000,
  "dims_contains": "plan=pro",
  "sort": "desc",
  "page_size": 100
}
```

### gRPC QueryStream
```protobuf
rpc QueryStream (QueryStreamRequest) returns (stream QueryStreamResponse);

message QueryStreamRequest {
  string tenant_id = 1;
  string session_id = 2;
  string event_name = 3;
  int64 from_ts = 4;
  int64 to_ts = 5;
  string dims_contains = 6;
  int32 page_size = 7;
}

message QueryStreamResponse {
  repeated Event events = 1;
}
```

---

## Live Streams (SSE)

All require admin authentication via cookie.

### Live Counters
```http
GET /live-counters?tenant_id=my-tenant
Cookie: admin_key=<admin_key>
Accept: text/event-stream
```

**Stream:**
```
data: {"count": 1234}

data: {"count": 1235}

data: {"count": 1236}
```

### Live Session
```http
GET /live-session?session_id=a1b2c3d4e5f6...
Cookie: admin_key=<admin_key>
Accept: text/event-stream
```

**Stream:**
```
data: {"event_count": 5, "last_seen": "2024-01-15T10:30:00Z"}

data: {"event_count": 6, "last_seen": "2024-01-15T10:31:00Z"}
```

### Live Events
```http
GET /live-events?session_id=a1b2c3d4e5f6...
Cookie: admin_key=<admin_key>
Accept: text/event-stream
```

**Stream:**
```
data: {"events": [{"EventID": "evt-001", "EventName": "click", ...}]}

data: {"events": [{"EventID": "evt-002", "EventName": "scroll", ...}]}
```

## Persistent analytics buckets

Bucket counts are persisted during ingestion for fast historical charts. Existing events are backfilled when the server starts.

```http
GET /analytics/buckets?tenant_id=my-tenant&event_name=page_view&bucket=hour&from_ts=1705276800000&to_ts=1705363200000
Cookie: admin_key=<admin_key>
```

`bucket` accepts `minute`, `10minute`, or `hour`. The response contains bucket start timestamps and counts:

```json
[
  {"bucket": 1705276800000, "count": 42},
  {"bucket": 1705280400000, "count": 57}
]
```

---

## Admin Endpoints

### Tenants
```http
GET    /tenants                    # List all
POST   /tenants                    # Create { "Name": "acme", "APIKey": "key" }
GET    /tenants/:id                # Get one
DELETE /tenants/:id                # Delete
POST   /tenants/:id/rotate-key     # Rotate API key
PUT    /tenants/:id/tier           # Set tier { "tier": "growth" }
```

### Schemas
```http
GET    /schemas?tenant_id=xxx      # List schemas
POST   /schemas                    # Register { "tenant_id": "xxx", "name": "click", "definition": {...} }
```

### Cohorts
```http
GET    /cohorts?tenant_id=xxx      # List
POST   /cohorts                    # Create { "tenant_id": "xxx", "name": "buyers", "event_name": "purchase", "min_count": 3 }
POST   /cohorts/:id/materialize    # Materialize membership
GET    /cohorts/:id/members        # Get member session IDs
```

### Alerts
```http
GET    /alerts?tenant_id=xxx       # List
POST   /alerts                     # Create rule
POST   /alerts/:id/enable          # Enable
POST   /alerts/:id/disable         # Disable
DELETE /alerts/:id                 # Delete
```

### Settings
```http
GET    /settings                   # Get current
PUT    /settings                   # Update { "retention_hours": 168, "rate_limit_per_second": 200 }
```

### Dashboard Cache
```http
GET    /dashboard-cache            # Cached metrics (refreshed every minute)
```

### Events Browser
```http
GET /events?tenant_id=xxx&event_name=click&from_ts=...&to_ts=...&sort=desc
```

### Replay
```http
POST /replay
{ "tenant_id": "xxx", "from_ts": 1705276800000, "to_ts": 1705363200000 }
```

---

## Event Schema

```json
{
  "event_id": "string (UUID)",
  "session_id": "string",
  "tenant_id": "string",
  "event_name": "string",
  "client_ts": 1705315800000,
  "ingest_ts": "2024-01-15T10:30:00Z",
  "payload": {},
  "dims": { "key": "value" },
  "source": "client|system",
  "is_aggregated": false,
  "aggregation_id": "",
  "experiment_id": "",
  "variant": ""
}
```

---

## Rate Limits

Tiers (requests/second):
- `free`: 50
- `starter`: 200
- `growth`: 1000
- `enterprise`: 5000

Default: 50. Configurable via `PUT /settings`.

---

## Error Responses

```json
{ "error": "description" }
```

| Status | Meaning |
|--------|---------|
| 400 | Bad request / validation error |
| 401 | Unauthorized (invalid/missing key) |
| 403 | Forbidden (admin endpoint with tenant key) |
| 404 | Not found |
| 429 | Rate limited |
| 500 | Internal error |
| 503 | Cache not ready (dashboard-cache) |
