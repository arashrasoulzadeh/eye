package handler

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
)

func secureKeyEqual(expected, actual string) bool {
	if expected == "" || actual == "" {
		return false
	}
	if len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

// AdminAuth wraps next with a check that the request carries the configured
// admin API key in the x-admin-key header, rejecting anything else with 401.
func AdminAuth(adminKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("x-admin-key")
		if !secureKeyEqual(adminKey, key) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// TenantOrAdminAuth allows a request through if it carries the admin key
// (x-admin-key), OR — for a POST whose JSON body has a "tenant_id" field —
// a tenant API key (x-api-key) that verify() confirms belongs to that
// tenant. This exists so a public, untrusted client (e.g. the CDN SDK
// running in a browser) can create a session for itself using only its own
// tenant API key, without ever holding the shared admin key: embedding that
// in client-side JS would defeat the whole point of an "admin" key. Routes
// that should stay admin-only (listing/deleting across tenants) are not
// wrapped with this — only session creation needs it.
func TenantOrAdminAuth(adminKey string, verify func(tenantID, apiKey string) bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if k := r.Header.Get("x-admin-key"); secureKeyEqual(adminKey, k) {
			next.ServeHTTP(w, r)
			return
		}

		if r.Method == http.MethodPost {
			apiKey := r.Header.Get("x-api-key")
			body, err := io.ReadAll(r.Body)
			if err == nil {
				var b struct {
					TenantID string `json:"tenant_id"`
				}
				if json.Unmarshal(body, &b) == nil && b.TenantID != "" && apiKey != "" && verify(b.TenantID, apiKey) {
					r.Body = io.NopCloser(bytes.NewReader(body))
					next.ServeHTTP(w, r)
					return
				}
			}
			// Restore the body even on the failure path, in case a future
			// caller wants to log/inspect it before the 401 response.
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// TenantIngestAuth authenticates a batch by resolving its session IDs to one
// tenant and verifying the caller's tenant API key. Admin callers remain
// allowed for replay/internal workflows. The body is restored before the
// wrapped handler runs.
func TenantIngestAuth(adminKey string, verify func(tenantID, apiKey string) bool, lookup func(sessionID string) (string, bool), next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secureKeyEqual(adminKey, r.Header.Get("x-admin-key")) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodPost || r.Header.Get("x-api-key") == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var envelope struct {
			Events []struct {
				SessionID string `json:"session_id"`
			} `json:"events"`
		}
		if json.Unmarshal(body, &envelope) != nil || len(envelope.Events) == 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tenantID := ""
		for _, event := range envelope.Events {
			resolved, ok := lookup(event.SessionID)
			if !ok || resolved == "" || (tenantID != "" && tenantID != resolved) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			tenantID = resolved
		}
		if !verify(tenantID, r.Header.Get("x-api-key")) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}
