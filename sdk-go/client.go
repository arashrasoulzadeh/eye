// Package eyesdk is a Go port of web-sdk's EyeClient: session management,
// a bounded ring buffer, and size/time-threshold flush batching against the
// eye collector's /sessions and /ingest REST endpoints.
package eyesdk

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Event mirrors web-sdk's Event shape (camelCase JSON fields match the TS SDK;
// the server's /ingest handler accepts arbitrary map[string]any per event, so
// field names here are chosen to match what IngestService expects via JSON tags).
type Event struct {
	EventID   string         `json:"event_id"`
	SessionID string         `json:"session_id"`
	EventName string         `json:"event_name"`
	ClientTS  int64          `json:"client_ts"`
	Payload   map[string]any `json:"payload"`
	Dims      map[string]any `json:"dims"`
}

// Options configures a Client. MaxBufferSize/BatchSize/FlushInterval default
// to the same values as web-sdk's EyeClient (1000 / 20 / 5s).
type Options struct {
	Endpoint      string
	APIKey        string
	TenantID      string
	MaxBufferSize int
	BatchSize     int
	FlushInterval time.Duration
	HTTPClient    *http.Client
}

const (
	defaultMaxBufferSize = 1000
	defaultBatchSize     = 20
	defaultFlushInterval = 5 * time.Second
)

// Client is a bounded, batching event buffer with a background flush timer.
type Client struct {
	opts Options
	http *http.Client

	mu        sync.Mutex
	buffer    []Event
	sessionID string

	stopOnce sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewClient constructs a Client and starts its background flush timer
// (unless FlushInterval is 0, which disables it, matching web-sdk's EyeClient).
func NewClient(opts Options) *Client {
	if opts.MaxBufferSize <= 0 {
		opts.MaxBufferSize = defaultMaxBufferSize
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaultBatchSize
	}
	if opts.FlushInterval == 0 {
		opts.FlushInterval = defaultFlushInterval
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}
	c := &Client{
		opts:   opts,
		http:   opts.HTTPClient,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	if opts.FlushInterval > 0 {
		go c.flushLoop()
	} else {
		close(c.doneCh)
	}
	return c
}

// Close stops the background flush timer. It does not flush remaining events.
func (c *Client) Close() {
	c.stopOnce.Do(func() { close(c.stopCh) })
	<-c.doneCh
}

func (c *Client) flushLoop() {
	defer close(c.doneCh)
	t := time.NewTicker(c.opts.FlushInterval)
	defer t.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-t.C:
			_ = c.Flush(context.Background())
		}
	}
}

// EnsureSession returns the current session id, creating one via POST
// /sessions if none exists yet (no persistence across process restarts —
// unlike web-sdk's localStorage, a Go process has no equivalent default
// store; callers that need persistence can seed SessionID themselves).
func (c *Client) EnsureSession(ctx context.Context) (string, error) {
	c.mu.Lock()
	sid := c.sessionID
	c.mu.Unlock()
	if sid != "" {
		return sid, nil
	}

	body, _ := json.Marshal(map[string]string{"tenant_id": c.opts.TenantID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.opts.Endpoint+"/sessions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.opts.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("eyesdk: create session failed: %s", resp.Status)
	}
	var out struct {
		ID string `json:"ID"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}

	c.mu.Lock()
	c.sessionID = out.ID
	c.mu.Unlock()
	return out.ID, nil
}

// SetSessionID lets a caller seed/restore a session id instead of creating a
// new one (the Go equivalent of web-sdk reading its id back out of localStorage).
func (c *Client) SetSessionID(id string) {
	c.mu.Lock()
	c.sessionID = id
	c.mu.Unlock()
}

func randomEventID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Track buffers an event. If the buffer is at MaxBufferSize, the oldest event
// is dropped (ring buffer). If BatchSize > 0 and the buffer reaches it, a
// background flush is triggered (errors are swallowed; the events remain
// buffered for the next attempt — same best-effort semantics as web-sdk).
func (c *Client) Track(name string, payload, dims map[string]any) error {
	c.mu.Lock()
	sid := c.sessionID
	if sid == "" {
		c.mu.Unlock()
		return fmt.Errorf("eyesdk: call EnsureSession first")
	}
	ev := Event{
		EventID:   randomEventID(),
		SessionID: sid,
		EventName: name,
		ClientTS:  time.Now().UnixMilli(),
		Payload:   payload,
		Dims:      dims,
	}
	if len(c.buffer) >= c.opts.MaxBufferSize {
		c.buffer = c.buffer[1:]
	}
	c.buffer = append(c.buffer, ev)
	shouldFlush := c.opts.BatchSize > 0 && len(c.buffer) >= c.opts.BatchSize
	c.mu.Unlock()

	if shouldFlush {
		go func() { _ = c.Flush(context.Background()) }()
	}
	return nil
}

// Flush POSTs all currently-buffered events to /ingest and clears the
// buffer on success. A no-op when the buffer is empty.
func (c *Client) Flush(ctx context.Context) error {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return nil
	}
	events := c.buffer
	c.buffer = nil
	c.mu.Unlock()

	body, err := json.Marshal(map[string]any{"events": events})
	if err != nil {
		c.requeue(events)
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.opts.Endpoint+"/ingest", bytes.NewReader(body))
	if err != nil {
		c.requeue(events)
		return err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.opts.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		c.requeue(events)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		c.requeue(events)
		return fmt.Errorf("eyesdk: flush failed: %s", resp.Status)
	}
	return nil
}

// requeue puts events back at the front of the buffer after a failed flush,
// trimmed to MaxBufferSize if needed.
func (c *Client) requeue(events []Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buffer = append(events, c.buffer...)
	if over := len(c.buffer) - c.opts.MaxBufferSize; over > 0 {
		c.buffer = c.buffer[over:]
	}
}

// BufferLen reports the current number of buffered (unflushed) events.
func (c *Client) BufferLen() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.buffer)
}

// PeekBuffer returns a copy of the currently buffered events, for tests and
// diagnostics (e.g. cross-SDK parity checks against web-sdk's EyeClient).
func (c *Client) PeekBuffer() []Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Event, len(c.buffer))
	copy(out, c.buffer)
	return out
}
