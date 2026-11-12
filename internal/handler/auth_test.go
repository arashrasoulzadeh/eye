package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTenantIngestAuth(t *testing.T) {
	lookup := func(sessionID string) (string, bool) {
		if sessionID == "s1" {
			return "tenant-a", true
		}
		if sessionID == "s2" {
			return "tenant-b", true
		}
		return "", false
	}
	verify := func(tenantID, key string) bool { return tenantID == "tenant-a" && key == "client-key" }
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := TenantIngestAuth("admin-key", verify, lookup, next)

	tests := []struct {
		name, key, body string
		want            int
	}{
		{"valid tenant key", "client-key", `{"events":[{"session_id":"s1"}]}`, http.StatusNoContent},
		{"missing key", "", `{"events":[{"session_id":"s1"}]}`, http.StatusUnauthorized},
		{"mixed tenants", "client-key", `{"events":[{"session_id":"s1"},{"session_id":"s2"}]}`, http.StatusUnauthorized},
		{"unknown session", "client-key", `{"events":[{"session_id":"missing"}]}`, http.StatusUnauthorized},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/ingest", strings.NewReader(tc.body))
			if tc.key != "" {
				req.Header.Set("x-api-key", tc.key)
			}
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			if res.Code != tc.want {
				t.Fatalf("status = %d, want %d", res.Code, tc.want)
			}
		})
	}
}

func TestAdminSessionAuthAcceptsAndRevokesSession(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AdminSession{}); err != nil {
		t.Fatal(err)
	}
	token, hash := adminSessionToken()
	if err := db.Create(&model.AdminSession{ID: "s1", TokenHash: hash, Email: "a@example.com", ExpiresAt: time.Now().Add(time.Hour).Unix()}).Error; err != nil {
		t.Fatal(err)
	}
	h := AdminSessionAuth(db, "legacy-key", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-admin-email") != "a@example.com" {
			t.Fatalf("email=%q", r.Header.Get("x-admin-email"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("x-admin-session", token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("session status=%d", w.Code)
	}
	if err := RevokeAdminSession(db, token); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("revoked status=%d", w.Code)
	}
}

func TestTenantIngestAuthAllowsAdmin(t *testing.T) {
	h := TenantIngestAuth("admin-key", func(string, string) bool { return false }, func(string) (string, bool) { return "", false }, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	req := httptest.NewRequest(http.MethodPost, "/ingest", strings.NewReader(`{"events":[]}`))
	req.Header.Set("x-admin-key", "admin-key")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
	}
}

func TestAdminRoleAuth(t *testing.T) {
	h := AdminRoleAuth("admin-key", func(email string) (string, bool) {
		if email == "viewer@example.com" {
			return "viewer", true
		}
		if email == "admin@example.com" {
			return "admin", true
		}
		return "", false
	}, []string{"owner", "admin"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, tc := range []struct {
		name, email string
		want        int
	}{
		{"shared key owner compatibility", "", http.StatusNoContent},
		{"active admin", "admin@example.com", http.StatusNoContent},
		{"viewer forbidden", "viewer@example.com", http.StatusForbidden},
		{"unknown identity forbidden", "unknown@example.com", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin-users", nil)
			req.Header.Set("x-admin-key", "admin-key")
			if tc.email != "" {
				req.Header.Set("x-admin-email", tc.email)
			}
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			if res.Code != tc.want {
				t.Fatalf("status=%d want=%d", res.Code, tc.want)
			}
		})
	}
}
