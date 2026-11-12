package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestObservabilityAddsTraceID(t *testing.T) {
	h := Observability(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) }))
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d", w.Code)
	}
	if len(w.Header().Get("X-Trace-ID")) != 32 {
		t.Fatalf("trace id=%q", w.Header().Get("X-Trace-ID"))
	}
}

func TestObservabilityPropagatesW3CTraceParent(t *testing.T) {
	h := Observability(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("traceparent"); got != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
			t.Fatalf("traceparent=%q", got)
		}
	}))
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got := w.Header().Get("traceparent"); got != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		t.Fatalf("response traceparent=%q", got)
	}
}
