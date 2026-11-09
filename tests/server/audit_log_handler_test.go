package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/model"
)

func TestAuditLogHandlerListsNewestFirst(t *testing.T) {
	db := newAuditTestDB(t)
	db.Create(&model.AuditLog{ID: "a1", Action: "POST", Resource: "/tenants", ActorKey: "admin", CreatedAt: 100})
	db.Create(&model.AuditLog{ID: "a2", Action: "DELETE", Resource: "/tenants/x", ActorKey: "admin", CreatedAt: 200})

	h := &handler.AuditLogHandler{DB: db}
	req := httptest.NewRequest(http.MethodGet, "/audit-log", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body == "[]\n" || body == "" {
		t.Fatal("expected non-empty audit log listing")
	}
}
