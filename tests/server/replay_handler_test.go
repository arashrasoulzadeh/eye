package test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/handler"
	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func TestReplayHandlerReplaysRange(t *testing.T) {
	db, r := newReplayTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	ingest := &service.IngestService{Repo: r}
	ingest.Ingest([]map[string]any{
		{"event_id": "e1", "session_id": "s1", "event_name": "click", "client_ts": float64(1000), "payload": map[string]any{}, "dims": map[string]any{}},
	})

	h := &handler.ReplayHandler{Svc: &service.ReplayService{DB: db, Ingest: ingest}}
	req := httptest.NewRequest(http.MethodPost, "/replay", bytes.NewBufferString(`{"tenant_id":"t1","from_ts":0,"to_ts":2000}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() == "" {
		t.Fatal("expected a json body")
	}
}

func TestReplayHandlerRejectsMissingTenant(t *testing.T) {
	db, r := newReplayTestDB(t)
	ingest := &service.IngestService{Repo: r}
	h := &handler.ReplayHandler{Svc: &service.ReplayService{DB: db, Ingest: ingest}}
	req := httptest.NewRequest(http.MethodPost, "/replay", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
