package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	h := HealthHandler{}
	t.Run("reports healthy without credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code != http.StatusOK || res.Body.String() != "{\"status\":\"ok\"}\n" {
			t.Fatalf("unexpected health response: %d %q", res.Code, res.Body.String())
		}
	})
	t.Run("rejects non GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusMethodNotAllowed)
		}
	})
}
