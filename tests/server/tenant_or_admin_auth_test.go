package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/handler"
)

func TestTenantOrAdminAuthAllowsAdminKey(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	verify := func(tenantID, apiKey string) bool { return false }
	h := handler.TenantOrAdminAuth("admin-secret", verify, inner)

	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"tenant_id":"t1"}`))
	req.Header.Set("x-admin-key", "admin-secret")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct admin key, got %d", w.Code)
	}
}

func TestTenantOrAdminAuthAllowsVerifiedTenantKey(t *testing.T) {
	var gotTenant, gotKey string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	verify := func(tenantID, apiKey string) bool {
		gotTenant, gotKey = tenantID, apiKey
		return tenantID == "t1" && apiKey == "correct-key"
	}
	h := handler.TenantOrAdminAuth("admin-secret", verify, inner)

	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"tenant_id":"t1"}`))
	req.Header.Set("x-api-key", "correct-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct tenant key, got %d: %s", w.Code, w.Body.String())
	}
	if gotTenant != "t1" || gotKey != "correct-key" {
		t.Fatalf("verify called with wrong args: tenant=%q key=%q", gotTenant, gotKey)
	}
}

func TestTenantOrAdminAuthRejectsWrongTenantKey(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	verify := func(tenantID, apiKey string) bool { return false }
	h := handler.TenantOrAdminAuth("admin-secret", verify, inner)

	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"tenant_id":"t1"}`))
	req.Header.Set("x-api-key", "wrong-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong tenant key, got %d", w.Code)
	}
}

func TestTenantOrAdminAuthRejectsNoCredentials(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	verify := func(tenantID, apiKey string) bool { return true }
	h := handler.TenantOrAdminAuth("admin-secret", verify, inner)

	req := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated GET, got %d", w.Code)
	}
}

func TestTenantOrAdminAuthBodyReachesHandlerAfterVerify(t *testing.T) {
	var sawBody string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		sawBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	})
	verify := func(tenantID, apiKey string) bool { return true }
	h := handler.TenantOrAdminAuth("admin-secret", verify, inner)

	body := `{"tenant_id":"t1"}`
	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(body))
	req.Header.Set("x-api-key", "any-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if sawBody != body {
		t.Fatalf("expected inner handler to see the original body %q, got %q", body, sawBody)
	}
}
