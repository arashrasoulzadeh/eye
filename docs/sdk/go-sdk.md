# Go SDK Documentation

Server-side Go client with API parity to the Web SDK.

## Installation

```bash
go get github.com/arashrasoulzadeh/eye/sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "github.com/arashrasoulzadeh/eye/sdk-go"
)

func main() {
    client := eyesdk.NewClient(eyesdk.Options{
        Endpoint: "http://localhost:8080",
        APIKey:   "tenant-api-key",
        TenantID: "my-tenant",
        Debug:    true,
    })
    defer client.Close()

    // Create/reuse session
    sessionID, err := client.EnsureSession(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Session: %s", sessionID)

    // Track events
    client.Track("purchase", map[string]any{
        "amount": 99.99,
        "currency": "USD",
    }, map[string]any{
        "plan": "pro",
    })

    // Identify user (adds user_id to subsequent events)
    client.Identify("user-123", map[string]any{
        "email": "user@example.com",
        "plan": "pro",
    })

    // Manual flush
    if err := client.Flush(context.Background()); err != nil {
        log.Printf("Flush error: %v", err)
    }
}
```

## Options

```go
type Options struct {
    Endpoint      string        // Required for flush
    APIKey        string        // Required for flush
    TenantID      string        // Required
    MaxBufferSize int           // Default: 1000
    BatchSize     int           // Default: 20
    FlushInterval time.Duration // Default: 5s (0 = disabled)
    HTTPClient    *http.Client  // Default: http.DefaultClient
    Debug         bool          // Enable logging
    Logger        *log.Logger   // Custom logger (default: log.Default())
}
```

## Methods

### `NewClient(opts Options) *Client`
Creates client and starts background flush timer (if `FlushInterval > 0`).

### `EnsureSession(ctx) (string, error)`
Creates session via `POST /sessions` if none exists. Returns session ID.
- Sends server device info: `{ "os": "linux", "arch": "amd64", "hostname": "..." }`
- No localStorage equivalent - call `SetSessionID()` to restore

### `SetSessionID(id string)`
Seeds/restores session ID (equivalent to reading from localStorage in web SDK).

### `Identify(userID string, traits map[string]any)`
Records user identity. Merges `user_id` and traits into subsequent event `dims`.

### `Track(name string, payload, dims map[string]any) error`
Buffers event. Ring buffer drops oldest at `MaxBufferSize`.
Auto-flushes in background when `BatchSize` reached.

### `Flush(ctx) error`
Posts buffered events to `/ingest`. Clears buffer on success.
Re-queues on failure (trimmed to `MaxBufferSize`).

### `Close()`
Stops background flush timer. Does NOT flush remaining events.

### `BufferLen() int`
Current buffered event count.

### `PeekBuffer() []Event`
Copy of buffered events (for testing/diagnostics).

## Event Structure

```go
type Event struct {
    EventID   string         `json:"event_id"`
    SessionID string         `json:"session_id"`
    EventName string         `json:"event_name"`
    ClientTS  int64          `json:"client_ts"`  // Unix milliseconds
    Payload   map[string]any `json:"payload"`
    Dims      map[string]any `json:"dims"`
}
```

Wire format matches web SDK (snake_case):
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

```go
client := eyesdk.NewClient(eyesdk.Options{
    Endpoint: "...",
    APIKey:   "...",
    TenantID: "t1",
    Debug:    true,
    Logger:   log.New(os.Stdout, "[eye] ", log.LstdFlags),
})
```

Output:
```
[eye] client created tenantId=t1 sessionId=
[eye] session established abc123
[eye] track name=purchase
[eye] flushing 1 events
[eye] flush complete
```

Validation errors returned instead of silently buffering.

## Cross-SDK Parity

| Feature | Web SDK | Go SDK |
|---------|---------|--------|
| Session persistence | localStorage | `SetSessionID()` |
| Ring buffer | ✓ | ✓ |
| Batch flush | ✓ | ✓ |
| Time-based flush | ✓ | ✓ |
| Identify/user_id | ✓ | ✓ |
| Device info | browser | server (os/arch/hostname) |
| Debug mode | ✓ | ✓ |
| Payload validation | ✓ | ✓ |

## Testing

```bash
cd sdk-go
go test -v
```

Tests cover:
- Ring buffer overflow behavior
- Batch flush triggering
- Session creation/reuse
- Identify merging
- Flush success/failure/requeue
- Parity with web-sdk (shared test vectors)