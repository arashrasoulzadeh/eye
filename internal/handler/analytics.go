package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

// FunnelHandler serves POST /analytics/funnel.
type FunnelHandler struct{ Svc *service.AnalyticsService }

type funnelRequest struct {
	TenantID string   `json:"tenant_id"`
	Steps    []string `json:"steps"`
	FromTS   *int64   `json:"from_ts"`
	ToTS     *int64   `json:"to_ts"`
}

func (h *FunnelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req funnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.TenantID == "" || len(req.Steps) == 0 {
		http.Error(w, "tenant_id and steps are required", http.StatusBadRequest)
		return
	}
	results, err := h.Svc.Funnel(req.TenantID, req.Steps, req.FromTS, req.ToTS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(results)
}

// RetentionHandler serves GET /analytics/retention?tenant_id=...&granularity=daily|weekly.
type RetentionHandler struct{ Svc *service.AnalyticsService }

// SessionEventsHandler serves GET /analytics/session-events?tenant_id=...
type SessionEventsHandler struct{ Svc *service.AnalyticsService }

type DeviceBreakdownHandler struct{ Svc *service.AnalyticsService }

type TopNHandler struct{ Svc *service.AnalyticsService }

type SlidingWindowHandler struct{ Svc *service.AnalyticsService }

type DistinctSessionsHandler struct{ Svc *service.AnalyticsService }

func (h *DistinctSessionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	count, err := h.Svc.DistinctSessions(tenantID, r.URL.Query().Get("event_name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]float64{"distinct_sessions": count})
}

func (h *SlidingWindowHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query()
	tenantID, eventName := query.Get("tenant_id"), query.Get("event_name")
	if tenantID == "" || eventName == "" {
		http.Error(w, "tenant_id and event_name are required", http.StatusBadRequest)
		return
	}
	windowSeconds := 60
	if raw := query.Get("window_seconds"); raw != "" {
		if _, err := fmt.Sscanf(raw, "%d", &windowSeconds); err != nil {
			http.Error(w, "window_seconds must be an integer", http.StatusBadRequest)
			return
		}
	}
	rows, err := h.Svc.SlidingWindowCounts(tenantID, eventName, int64(windowSeconds), nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(rows)
}

func (h *TopNHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if _, err := fmt.Sscanf(raw, "%d", &limit); err != nil {
			http.Error(w, "limit must be an integer", http.StatusBadRequest)
			return
		}
	}
	rows, err := h.Svc.TopN(tenantID, r.URL.Query().Get("event_name"), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rows)
}

func (h *DeviceBreakdownHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	rows, err := h.Svc.DeviceBreakdown(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rows)
}

func (h *SessionEventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	rows, err := h.Svc.EventsPerSession(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rows)
}

func (h *RetentionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	granularity := r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = "daily"
	}
	if granularity != "daily" && granularity != "weekly" {
		http.Error(w, "granularity must be daily or weekly", http.StatusBadRequest)
		return
	}
	rows, err := h.Svc.Retention(tenantID, granularity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rows)
}
