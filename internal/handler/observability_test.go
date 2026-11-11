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
