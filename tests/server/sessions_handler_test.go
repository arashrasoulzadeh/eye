package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eye/eye/internal/handler"
)

func TestSessionsHandlerCreateAndList(t *testing.T) {
	db := newEventsTestDB(t)
	h := &handler.SessionsHandler{DB: db}

	body := strings.NewReader(`{"tenant_id":"t1"}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 creating session, got %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID       string `json:"ID"`
		TenantID string `json:"TenantID"`
	}
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.TenantID != "t1" {
		t.Fatalf("expected a generated session id for tenant t1, got %+v", created)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	var list []map[string]any
	if err := json.NewDecoder(w2.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
}
