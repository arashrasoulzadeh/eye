package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/model"
)

func newAuditTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAuditMiddlewareLogsMutatingAdminRequests(t *testing.T) {
	db := newAuditTestDB(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) })

	chain := handler.AdminAuth("secret", handler.Audit(db, inner))

	req := httptest.NewRequest(http.MethodPost, "/tenants", nil)
	req.Header.Set("x-admin-key", "secret")
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected inner handler's 201, got %d", rec.Code)
	}

	var entries []model.AuditLog
	db.Find(&entries)
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].Action != http.MethodPost || entries[0].Resource != "/tenants" {
		t.Fatalf("unexpected audit entry: %+v", entries[0])
	}
	if entries[0].StatusCode != http.StatusCreated {
		t.Fatalf("expected recorded status 201, got %d", entries[0].StatusCode)
	}
	if entries[0].ActorKey != "admin" {
		t.Fatalf("expected ActorKey 'admin' (single shared key, no per-user identity), got %q", entries[0].ActorKey)
	}
}

func TestAuditMiddlewareDoesNotLogGetRequests(t *testing.T) {
	db := newAuditTestDB(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	chain := handler.AdminAuth("secret", handler.Audit(db, inner))

	req := httptest.NewRequest(http.MethodGet, "/tenants", nil)
	req.Header.Set("x-admin-key", "secret")
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected GET requests not to be audit-logged, got %d entries", count)
	}
}

func TestAuditMiddlewareDoesNotLogFailedAuth(t *testing.T) {
	db := newAuditTestDB(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) })
	chain := handler.AdminAuth("secret", handler.Audit(db, inner))

	req := httptest.NewRequest(http.MethodPost, "/tenants", nil)
	req.Header.Set("x-admin-key", "wrong-key")
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected a failed-auth request not to be audit-logged, got %d entries", count)
	}
}
