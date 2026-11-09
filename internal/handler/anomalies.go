package handler

import (
	"encoding/json"
	"net/http"

	"github.com/eye/eye/internal/service"
)

// AnomaliesHandler exposes the Phase 8 z-score anomaly detector's current
// findings: which (tenant, event_name) pairs' latest sampled bucket is more
// than AnomalyService.ZThreshold standard deviations above their own recent
// rolling mean.
type AnomaliesHandler struct{ Svc *service.AnomalyService }

func (h *AnomaliesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
