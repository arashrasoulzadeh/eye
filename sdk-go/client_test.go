package eyesdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func nil2() context.Context { return context.Background() }

func newTestClient(t *testing.T, srv *httptest.Server, opts Options) *Client {
	t.Helper()
	opts.Endpoint = srv.URL
	c := NewClient(opts)
	t.Cleanup(c.Close)
	return c
}

func TestRingBufferEvictsOldestAtCapacity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"accepted":0,"rejected":0}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, Options{TenantID: "t1", MaxBufferSize: 3, BatchSize: 0, FlushInterval: time.Hour})
	c.SetSessionID("sess-1")

	for i := 0; i < 5; i++ {
		if err := c.Track("ev", map[string]any{"i": i}, nil); err != nil {
			t.Fatalf("track: %v", err)
		}
	}

	if got := c.BufferLen(); got != 3 {
		t.Fatalf("BufferLen = %d, want 3", got)
	}

	c.mu.Lock()
	first := c.buffer[0].Payload["i"]
	last := c.buffer[len(c.buffer)-1].Payload["i"]
	c.mu.Unlock()

	// Oldest two (i=0,1) should have been evicted; remaining should be 2,3,4 in order.
	if fv, ok := first.(int); !ok || fv != 2 {
		t.Fatalf("oldest remaining payload.i = %v, want 2", first)
	}
	if lv, ok := last.(int); !ok || lv != 4 {
		t.Fatalf("newest payload.i = %v, want 4", last)
	}
}

func TestFlushTriggeredBySizeThreshold(t *testing.T) {
	var received int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Events []Event `json:"events"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		atomic.AddInt32(&received, int32(len(body.Events)))
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"accepted":1,"rejected":0}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, Options{TenantID: "t1", MaxBufferSize: 100, BatchSize: 2, FlushInterval: time.Hour})
	c.SetSessionID("sess-1")

	_ = c.Track("a", nil, nil)
	_ = c.Track("b", nil, nil) // hits BatchSize=2, triggers async flush

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&received) == 2 && c.BufferLen() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("size-threshold flush did not complete: received=%d bufferLen=%d", received, c.BufferLen())
}

func TestFlushTriggeredByTimeThreshold(t *testing.T) {
	var received int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Events []Event `json:"events"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		atomic.AddInt32(&received, int32(len(body.Events)))
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"accepted":1,"rejected":0}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, Options{TenantID: "t1", MaxBufferSize: 100, BatchSize: 0, FlushInterval: 30 * time.Millisecond})
	c.SetSessionID("sess-1")
	_ = c.Track("a", nil, nil)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&received) == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("time-threshold flush did not fire: received=%d", received)
}

func TestEnsureSessionCreatesAndCaches(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		var body struct {
			TenantID string `json:"tenant_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.TenantID != "t1" {
			t.Errorf("tenant_id = %q, want t1", body.TenantID)
		}
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(map[string]string{"ID": "sess-abc"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, Options{TenantID: "t1", FlushInterval: time.Hour})

	id, err := c.EnsureSession(nil2())
	if err != nil {
		t.Fatalf("EnsureSession: %v", err)
	}
	if id != "sess-abc" {
		t.Fatalf("session id = %q, want sess-abc", id)
	}

	id2, err := c.EnsureSession(nil2())
	if err != nil {
		t.Fatalf("EnsureSession (cached): %v", err)
	}
	if id2 != "sess-abc" {
		t.Fatalf("cached session id = %q, want sess-abc", id2)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected EnsureSession to hit the server once, got %d calls", calls)
	}
}

func TestFlushIntegrationAgainstHTTPTestServer(t *testing.T) {
	var gotEvents []Event
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Events []Event `json:"events"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotEvents = body.Events
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"accepted":2,"rejected":0}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, Options{TenantID: "t1", FlushInterval: time.Hour})
	c.SetSessionID("sess-1")
	_ = c.Track("page_view", map[string]any{"path": "/"}, map[string]any{"platform": "server"})
	_ = c.Track("click", map[string]any{"btn": "signup"}, nil)

	if err := c.Flush(nil2()); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if len(gotEvents) != 2 {
		t.Fatalf("server received %d events, want 2", len(gotEvents))
	}
	if gotEvents[0].EventName != "page_view" || gotEvents[1].EventName != "click" {
		t.Fatalf("unexpected event order/names: %+v", gotEvents)
	}
	if c.BufferLen() != 0 {
		t.Fatalf("buffer should be empty after successful flush, got %d", c.BufferLen())
	}
}
