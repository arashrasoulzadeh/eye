package handler

import (
	"encoding/json"
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
