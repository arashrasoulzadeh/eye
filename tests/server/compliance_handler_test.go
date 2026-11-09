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

func TestComplianceHandlerErasesSession(t *testing.T) {
	db := newComplianceTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click"})

	h := &handler.ComplianceHandler{Svc: &service.ComplianceService{DB: db}}
	req := httptest.NewRequest(http.MethodDelete, "/compliance/erase", bytes.NewBufferString(`{"session_id":"s1"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var count int64
	db.Model(&model.Session{}).Where("id = ?", "s1").Count(&count)
	if count != 0 {
		t.Fatalf("expected session erased, found %d", count)
	}
}

func TestComplianceHandlerUnknownSessionIs404(t *testing.T) {
	db := newComplianceTestDB(t)
	h := &handler.ComplianceHandler{Svc: &service.ComplianceService{DB: db}}
	req := httptest.NewRequest(http.MethodDelete, "/compliance/erase", bytes.NewBufferString(`{"session_id":"nope"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestComplianceHandlerRejectsNonDelete(t *testing.T) {
	db := newComplianceTestDB(t)
	h := &handler.ComplianceHandler{Svc: &service.ComplianceService{DB: db}}
	req := httptest.NewRequest(http.MethodGet, "/compliance/erase", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
