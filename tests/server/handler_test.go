package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
)

func newAdminHandlerTestDeps(t *testing.T) *service.AdminService {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	if err := r.Migrate(); err != nil {
		t.Fatal(err)
	}
	return &service.AdminService{Repo: r, Salt: "test-salt"}
}

func TestTenantsHandlerRotateKey(t *testing.T) {
	as := newAdminHandlerTestDeps(t)
	tn, err := as.CreateTenant("acme", "key123")
	if err != nil {
		t.Fatal(err)
	}

	h := &handler.TenantsHandler{Svc: as}
	req := httptest.NewRequest(http.MethodPost, "/tenants/"+tn.ID+"/rotate-key", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.APIKey == "" || resp.APIKey == "key123" {
		t.Fatalf("expected a freshly rotated plaintext key, got %q", resp.APIKey)
	}
}

func TestSchemasHandlerRegisterAndList(t *testing.T) {
	as := newAdminHandlerTestDeps(t)
	tn, _ := as.CreateTenant("acme", "key123")

	h := &handler.SchemasHandler{Svc: as}

	body := strings.NewReader(`{"tenant_id":"` + tn.ID + `","name":"click","definition":"{}"}`)
	req := httptest.NewRequest(http.MethodPost, "/schemas", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on register, got %d: %s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/schemas?tenant_id="+tn.ID, nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	var list []map[string]any
	if err := json.NewDecoder(w2.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 registered schema, got %d", len(list))
	}
}

func TestAdminAuthMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	mw := handler.AdminAuth("secret-admin-key", inner)

	req := httptest.NewRequest(http.MethodGet, "/tenants", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no key, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/tenants", nil)
	req2.Header.Set("x-admin-key", "wrong")
	w2 := httptest.NewRecorder()
	mw.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong key, got %d", w2.Code)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/tenants", nil)
	req3.Header.Set("x-admin-key", "secret-admin-key")
	w3 := httptest.NewRecorder()
	mw.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct key, got %d", w3.Code)
	}
}
