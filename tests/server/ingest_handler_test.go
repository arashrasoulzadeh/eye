package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
)

func newIngestHandlerTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	if err := r.Migrate(); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestIngestHandlerAcceptsValidEvent(t *testing.T) {
	db := newIngestHandlerTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	h := &handler.IngestHandler{Svc: &service.IngestService{Repo: repository.New(db)}}

	body, _ := json.Marshal(map[string]any{
		"events": []map[string]any{
			{"event_id": "e1", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"},
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
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestIngestHandlerRejectsUnknownSession(t *testing.T) {
	db := newIngestHandlerTestDB(t)
	h := &handler.IngestHandler{Svc: &service.IngestService{Repo: repository.New(db)}}

	body, _ := json.Marshal(map[string]any{
		"events": []map[string]any{
			{"event_id": "e1", "event_name": "click", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "missing"},
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
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestIngestHandlerRejectsMalformedJSON(t *testing.T) {
	db := newIngestHandlerTestDB(t)
	h := &handler.IngestHandler{Svc: &service.IngestService{Repo: repository.New(db)}}

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
