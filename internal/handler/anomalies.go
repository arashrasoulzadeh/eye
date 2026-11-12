package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

// AnomaliesHandler exposes the Phase 8 z-score anomaly detector's current
// findings: which (tenant, event_name) pairs' latest sampled bucket is more
// than AnomalyService.ZThreshold standard deviations above their own recent
// rolling mean.
type AnomaliesHandler struct{ Svc *service.AnomalyService }

func (h *AnomaliesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("page") == "true" {
		limit, offset := queryInt(r, "limit", 10), queryInt(r, "offset", 0)
		anomalies, total, err := h.Svc.CurrentAnomaliesPage(r.URL.Query().Get("filter"), r.URL.Query().Get("dir"), limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"items": anomalies, "total": total, "limit": limit, "offset": offset})
		return
	}
	anomalies, err := h.Svc.CurrentAnomalies()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if anomalies == nil {
		anomalies = []service.Anomaly{}
	}
	json.NewEncoder(w).Encode(anomalies)
}
