package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dto "github.com/prometheus/client_model/go"

	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/metrics"
	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
)

func readCounter(t *testing.T, label string) float64 {
	got := &dto.Metric{}
	if err := metrics.IngestTotal.WithLabelValues(label).Write(got); err != nil {
		t.Fatal(err)
	}
	return got.GetCounter().GetValue()
}

func readErrorCounter(t *testing.T, reason string) float64 {
	got := &dto.Metric{}
	if err := metrics.IngestErrorsTotal.WithLabelValues(reason).Write(got); err != nil {
		t.Fatal(err)
	}
	return got.GetCounter().GetValue()
}

// TestIngestHandlerRecordsAcceptedRejectedMetrics ingests a mix of a valid
// and an invalid event through the real IngestHandler and asserts the
// Phase 9 Prometheus ingest-rate counters moved by exactly the right amount.
func TestIngestHandlerRecordsAcceptedRejectedMetrics(t *testing.T) {
	db := newIngestHandlerTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	h := &handler.IngestHandler{Svc: &service.IngestService{Repo: repository.New(db)}}

	before := readCounter(t, "accepted")
	beforeRejected := readCounter(t, "rejected")

	body, _ := json.Marshal(map[string]any{
		"events": []map[string]any{
			{"event_id": "mx1", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"},
			{"event_id": "mx2", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "missing"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := readCounter(t, "accepted") - before; got != 1 {
		t.Fatalf("accepted counter delta = %v, want 1", got)
	}
	if got := readCounter(t, "rejected") - beforeRejected; got != 1 {
		t.Fatalf("rejected counter delta = %v, want 1", got)
	}
}

// TestIngestHandlerRecordsDecodeErrorMetric asserts malformed request bodies
// increment the dedicated ingest_errors_total series rather than silently
// disappearing before any metric is touched.
func TestIngestHandlerRecordsDecodeErrorMetric(t *testing.T) {
	db := newIngestHandlerTestDB(t)
	h := &handler.IngestHandler{Svc: &service.IngestService{Repo: repository.New(db)}}

	before := readErrorCounter(t, "decode")

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := readErrorCounter(t, "decode") - before; got != 1 {
		t.Fatalf("decode error counter delta = %v, want 1", got)
	}
}
