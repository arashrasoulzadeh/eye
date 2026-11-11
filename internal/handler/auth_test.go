package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
