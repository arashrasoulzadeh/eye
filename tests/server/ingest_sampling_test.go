package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
)

// fakeSampler always returns a fixed decision, for deterministic handler tests
// without depending on SamplingController's own (already separately tested)
// randomness.
type fakeSampler struct{ accept bool }

func (f fakeSampler) Accept() bool { return f.accept }

func TestIngestHandlerDropsEventsWhenSamplerRejects(t *testing.T) {
	db := newIngestHandlerTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	h := &handler.IngestHandler{
		Svc:     &service.IngestService{Repo: repository.New(db)},
		Sampler: fakeSampler{accept: false},
	}

	body, _ := json.Marshal(map[string]any{
		"events": []map[string]any{
			{"event_id": "sm1", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["accepted"] != 0 || resp["rejected"] != 1 {
		t.Fatalf("expected the sampled-out event to count as rejected, got %+v", resp)
	}

	var count int64
	db.Model(&model.Event{}).Where("event_id = ?", "sm1").Count(&count)
	if count != 0 {
		t.Fatal("sampled-out event should never reach storage")
	}
}

func TestIngestHandlerKeepsEventsWhenSamplerAccepts(t *testing.T) {
	db := newIngestHandlerTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	h := &handler.IngestHandler{
		Svc:     &service.IngestService{Repo: repository.New(db)},
		Sampler: fakeSampler{accept: true},
	}

	body, _ := json.Marshal(map[string]any{
		"events": []map[string]any{
			{"event_id": "sm2", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["accepted"] != 1 || resp["rejected"] != 0 {
		t.Fatalf("expected the accepted event to pass through, got %+v", resp)
	}
}

func TestIngestHandlerWithoutSamplerKeepsAllEvents(t *testing.T) {
	db := newIngestHandlerTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	h := &handler.IngestHandler{Svc: &service.IngestService{Repo: repository.New(db)}}

	body, _ := json.Marshal(map[string]any{
		"events": []map[string]any{
			{"event_id": "sm3", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["accepted"] != 1 {
		t.Fatalf("nil Sampler must not drop anything, got %+v", resp)
	}
}
