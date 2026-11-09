package eyesdk

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDebugModeLogsTrackAndFlush(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"accepted":1,"rejected":0}`))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	c := newTestClient(t, srv, Options{
		TenantID: "t1", MaxBufferSize: 10, BatchSize: 0, FlushInterval: time.Hour,
		Debug: true, Logger: log.New(&buf, "", 0),
	})
	c.SetSessionID("s1")

	if err := c.Track("click", map[string]any{"x": 1}, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.Flush(nil2()); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "track name=click") {
		t.Errorf("expected track log line, got: %s", out)
	}
	if !strings.Contains(out, "flushing 1 events") {
		t.Errorf("expected flush log line, got: %s", out)
	}
}

func TestDebugModeRejectsEmptyEventName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newTestClient(t, srv, Options{
		TenantID: "t1", MaxBufferSize: 10, BatchSize: 0, FlushInterval: time.Hour, Debug: true,
	})
	c.SetSessionID("s1")

	if err := c.Track("", nil, nil); err == nil {
		t.Error("expected error for empty event name in debug mode")
	}
	if c.BufferLen() != 0 {
		t.Error("invalid event should not have been buffered")
	}
}

func TestNonDebugModeDoesNotValidate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newTestClient(t, srv, Options{
		TenantID: "t1", MaxBufferSize: 10, BatchSize: 0, FlushInterval: time.Hour,
	})
	c.SetSessionID("s1")

	if err := c.Track("", nil, nil); err != nil {
		t.Errorf("non-debug mode should not validate event shape, got error: %v", err)
	}
	if c.BufferLen() != 1 {
		t.Error("event should have been buffered even with an empty name outside debug mode")
	}
}
