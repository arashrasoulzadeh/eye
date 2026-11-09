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

// fakeLimiter always returns a fixed decision, for deterministic handler
// tests without depending on LiveRateLimiter's own timing.
type fakeLimiter struct{ allow bool }

func (f fakeLimiter) Allow() bool { return f.allow }

func TestIngestHandlerRejectsWhenLimiterDenies(t *testing.T) {
	db := newIngestHandlerTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	h := &handler.IngestHandler{
		Svc:     &service.IngestService{Repo: repository.New(db)},
		Limiter: fakeLimiter{allow: false},
	}

	body, _ := json.Marshal(map[string]any{
		"events": []map[string]any{
			{"event_id": "rl1", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestIngestHandlerAllowsWhenLimiterPermits(t *testing.T) {
	db := newIngestHandlerTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	h := &handler.IngestHandler{
		Svc:     &service.IngestService{Repo: repository.New(db)},
		Limiter: fakeLimiter{allow: true},
	}

	body, _ := json.Marshal(map[string]any{
		"events": []map[string]any{
			{"event_id": "rl2", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["accepted"] != 1 {
		t.Fatalf("expected 1 accepted event, got %+v", resp)
	}
}
