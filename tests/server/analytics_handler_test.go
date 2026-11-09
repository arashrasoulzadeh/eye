package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/service"
)

func TestFunnelHandlerReturnsConversionRates(t *testing.T) {
	db := newAnalyticsTestDB(t)
	mustInsertEvent(t, db, "a1", "sessA", "t1", "page_view", 100)
	mustInsertEvent(t, db, "a2", "sessA", "t1", "signup", 200)

	h := &handler.FunnelHandler{Svc: &service.AnalyticsService{DB: db}}
	body := `{"tenant_id":"t1","steps":["page_view","signup"]}`
	req := httptest.NewRequest(http.MethodPost, "/analytics/funnel", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var results []service.FunnelStepResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(results) != 2 || results[0].Sessions != 1 || results[1].Sessions != 1 {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestFunnelHandlerRequiresTenantAndSteps(t *testing.T) {
	db := newAnalyticsTestDB(t)
	h := &handler.FunnelHandler{Svc: &service.AnalyticsService{DB: db}}
	req := httptest.NewRequest(http.MethodPost, "/analytics/funnel", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRetentionHandlerDefaultsToDaily(t *testing.T) {
	db := newAnalyticsTestDB(t)
	mustInsertEvent(t, db, "e1", "s1", "t1", "page_view", 1000*86400*1000)

	h := &handler.RetentionHandler{Svc: &service.AnalyticsService{DB: db}}
	req := httptest.NewRequest(http.MethodGet, "/analytics/retention?tenant_id=t1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var rows []service.RetentionRow
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 || rows[0].CohortSize != 1 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestRetentionHandlerRejectsBadGranularity(t *testing.T) {
	db := newAnalyticsTestDB(t)
	h := &handler.RetentionHandler{Svc: &service.AnalyticsService{DB: db}}
	req := httptest.NewRequest(http.MethodGet, "/analytics/retention?tenant_id=t1&granularity=monthly", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestExportHandlerJSONScopedToTenant(t *testing.T) {
	db := newAnalyticsTestDB(t)
	mustInsertEvent(t, db, "a1", "sessA", "t1", "page_view", 100)
	mustInsertEvent(t, db, "x1", "sessX", "t2", "page_view", 100)

	h := &handler.ExportHandler{Svc: &service.QueryService{DB: db}}
	req := httptest.NewRequest(http.MethodGet, "/export/events?tenant_id=t1&format=json", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var events []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(events) != 1 || events[0]["EventID"] != "a1" {
		t.Fatalf("expected only t1's event, got %+v", events)
	}
}

func TestExportHandlerCSVHasHeaderAndRows(t *testing.T) {
	db := newAnalyticsTestDB(t)
	mustInsertEvent(t, db, "a1", "sessA", "t1", "page_view", 100)

	h := &handler.ExportHandler{Svc: &service.QueryService{DB: db}}
	req := httptest.NewRequest(http.MethodGet, "/export/events?tenant_id=t1&format=csv", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Fatalf("expected text/csv content type, got %s", ct)
	}
	lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 1 header + 1 data row, got %d lines: %v", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "event_id,session_id,tenant_id") {
		t.Fatalf("unexpected header: %s", lines[0])
	}
	if !strings.Contains(lines[1], "a1,sessA,t1,page_view") {
		t.Fatalf("unexpected data row: %s", lines[1])
	}
}

func TestExportHandlerRejectsMissingTenant(t *testing.T) {
	db := newAnalyticsTestDB(t)
	h := &handler.ExportHandler{Svc: &service.QueryService{DB: db}}
	req := httptest.NewRequest(http.MethodGet, "/export/events?format=json", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
