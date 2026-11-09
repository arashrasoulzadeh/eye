package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eye/eye/internal/handler"
)

func TestSessionsHandlerCreateAndList(t *testing.T) {
	db := newEventsTestDB(t)
	h := &handler.SessionsHandler{DB: db}

	body := strings.NewReader(`{"tenant_id":"t1"}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 creating session, got %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID       string `json:"ID"`
		TenantID string `json:"TenantID"`
	}
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.TenantID != "t1" {
		t.Fatalf("expected a generated session id for tenant t1, got %+v", created)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	var list []map[string]any
	if err := json.NewDecoder(w2.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
}

func TestSessionsHandlerCapturesUserAgentIPAndDeviceInfo(t *testing.T) {
	db := newEventsTestDB(t)
	h := &handler.SessionsHandler{DB: db}

	body := strings.NewReader(`{"tenant_id":"t1","device":{"platform":"MacIntel","language":"en-US","screen_width":1920,"screen_height":1080,"timezone":"Asia/Tehran"}}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions", body)
	req.Header.Set("User-Agent", "Mozilla/5.0 (test-agent)")
	req.RemoteAddr = "203.0.113.7:54321"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var created struct {
		UserAgent  string `json:"UserAgent"`
		IP         string `json:"IP"`
		DeviceInfo string `json:"DeviceInfo"`
	}
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.UserAgent != "Mozilla/5.0 (test-agent)" {
		t.Errorf("expected UserAgent captured from the request header, got %q", created.UserAgent)
	}
	if created.IP != "203.0.113.7" {
		t.Errorf("expected IP captured (port stripped) from RemoteAddr, got %q", created.IP)
	}
	var device map[string]any
	if err := json.Unmarshal([]byte(created.DeviceInfo), &device); err != nil {
		t.Fatalf("DeviceInfo should be valid JSON, got %q: %v", created.DeviceInfo, err)
	}
	if device["platform"] != "MacIntel" || device["timezone"] != "Asia/Tehran" {
		t.Errorf("expected client-supplied device fields preserved, got %+v", device)
	}
}

func TestSessionsHandlerPrefersXForwardedForOverRemoteAddr(t *testing.T) {
	db := newEventsTestDB(t)
	h := &handler.SessionsHandler{DB: db}

	body := strings.NewReader(`{"tenant_id":"t1"}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions", body)
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.1")
	req.RemoteAddr = "10.0.0.1:443"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var created struct {
		IP string `json:"IP"`
	}
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.IP != "198.51.100.1" {
		t.Errorf("expected the leftmost X-Forwarded-For entry, got %q", created.IP)
	}
}
