# Web SDK Documentation

TypeScript client for browser-based event tracking with automatic batching, session persistence, and real-time features.

## Installation

```bash
cd web-sdk
npm install
```

## Quick Start

```typescript
import { EyeClient } from 'eye-sdk';

const eye = new EyeClient({
  tenantId: 'my-tenant',
  endpoint: 'https://api.example.com',
  apiKey: 'tenant-api-key',
  debug: true  // enables console logging
});

// Create/reuse session
await eye.ensureSession();

// Track events
eye.track('button_click', { button: 'signup' }, { plan: 'pro' });
eye.track('page_view', { url: '/pricing' });

// Identify user (adds user_id to all subsequent events)
eye.identify('user-123', { plan: 'pro', source: 'organic' });

// Manual flush
await eye.flush();

// Cleanup
eye.destroy();
```

## CDN Build

```bash
npm run build:cdn
# Outputs: dist/eye.min.js, dist/eye.min.js.map
```

```html
<script src="https://cdn.jsdelivr.net/gh/arashrasoulzadeh/eye@cdn/eye.min.js"></script>
<script>
  Eye.init({
    apiKey: '<tenant-api-key>',
    tenant: '<tenant-id>',
    endpoint: 'https://api.example.com'
  });
  
  Eye.track('button_click', { button: 'signup' });
  Eye.identify('user-123', { plan: 'pro' });
</script>
```

### Auto-Track Options

The CDN build auto-tracks these events by default (toggleable in `init()`):

```typescript
Eye.init({
  apiKey: '...',
  tenant: '...',
  endpoint: '...',
  autoTrack: {
    pageView: true,      // track page views
    click: true,         // track clicks
    formSubmit: true,    // track form submissions
    scrollDepth: true,   // track scroll milestones (25%, 50%, 75%, 100%)
    error: true,         // track JS errors
    unhandledRejection: true,
    performance: true,   // track navigation timing
    visibilityChange: true,
    beforeUnload: true
  }
});
```

## EyeClient API

### Constructor

```typescript
new EyeClient(options: EyeClientOptions)
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `tenantId` | string | required | Tenant identifier |
| `endpoint` | string | undefined | API base URL (required for flush) |
| `apiKey` | string | undefined | Tenant API key (required for flush) |
| `storageKey` | string | 'eye_session_id' | localStorage key for session |
| `maxBufferSize` | number | 1000 | Ring buffer capacity |
| `batchSize` | number | 20 | Auto-flush when buffer reaches this |
| `flushIntervalMs` | number | 5000 | Auto-flush interval (0 = disabled) |
| `debug` | boolean | false | Enable console logging |

### Methods

#### `ensureSession(endpoint?, apiKey?, { forceNew }?)`
Creates or reuses a session via `POST /sessions`. Persists session ID in localStorage.

```typescript
await eye.ensureSession();  // uses constructor endpoint/apiKey
await eye.ensureSession('https://api.example.com', 'key', { forceNew: true });
```

#### `getSessionId(): string | null`
Returns current session ID or null.

#### `track(name: string, props?: Record<string, any>, payload?: Record<string, any>)`
Queues an event. Auto-flushes if batch size reached.

```typescript
eye.track('purchase', { plan: 'pro' }, { amount: 99.99, currency: 'USD' });
```

- `name`: Event name (required)
- `props`: Dimensions (merged into `dims`, all values stringified)
- `payload`: Full payload object

#### `identify(userId: string, traits: Record<string, any>)`
Sets user identity. Merges `user_id` and traits into every subsequent event's `dims`.

```typescript
eye.identify('user-123', { email: 'user@example.com', plan: 'pro' });
```

#### `flush(endpointOverride?, apiKeyOverride?): Promise<void>`
Sends all buffered events to `/ingest`. Clears buffer on success.

#### `destroy()`
Stops the auto-flush timer.

#### `BufferLen(): number`
Returns current buffer length (for debugging).

#### `PeekBuffer(): Event[]`
Returns copy of buffered events (for debugging).

## Device Info Collection

Automatically collected (server can't derive these):
- `screenWidth`, `screenHeight`
- `viewportWidth`, `viewportHeight`
- `devicePixelRatio`
- `platform`, `language`, `languages`
- `timezone`

Sent with session creation, not with each event.

## Event Shape

```typescript
interface Event {
  eventId: string;      // UUID
  eventName: string;
  clientTs: number;     // Date.now()
  payload: Record<string, any>;
  dims?: Record<string, string>;
  sessionId: string;
}
```

Wire format (snake_case sent to server):
```json
{
  "events": [{
    "session_id": "...",
    "event_id": "...",
    "event_name": "...",
    "client_ts": 1705315800000,
    "payload": {},
    "dims": {}
  }]
}
```

## Debug Mode

```typescript
const eye = new EyeClient({ 
  tenantId: 't1', 
  endpoint: '...', 
  apiKey: '...',
  debug: true 
});
```

Console output:
```
[eye] client created { tenantId: 't1', sessionId: 'abc' }
[eye] session established abc
[eye] track { name: 'click', props: { btn: 'buy' }, payload: { amount: 10 } }
[eye] flushing 1 events
[eye] flush complete
```

Validation errors throw instead of silently buffering.

## Testing

```bash
cd web-sdk
npm test  # vitest
```

Tests cover:
- Ring buffer behavior (overflow drops oldest)
- Batch flush triggering
- Session persistence (localStorage)
- Identify merging into dims
- Payload validation
- Wire format conversion

## TypeScript

Full type definitions included. Import types:

```typescript
import { EyeClient, EyeClientOptions, Event } from 'eye-sdk';
```