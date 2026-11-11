package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arashrasoulzadeh/eye/internal/metrics"
)

// SLOHandler exposes the process availability SLO and remaining error budget.
// The counters intentionally cover requests observed since process start; a
// Prometheus time-window query remains the source for long-term reporting.
type SLOHandler struct{ Target float64 }

func (h SLOHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	target := h.Target
	if target <= 0 || target >= 1 {
		target = 0.999
	}
	total, errors := metrics.HTTPRequestSnapshot()
	availability := 1.0
	if total > 0 {
		availability = float64(total-errors) / float64(total)
	}
	budget := float64(total) * (1 - target)
	remaining := budget - float64(errors)
	if remaining < 0 {
		remaining = 0
	}
	json.NewEncoder(w).Encode(map[string]any{"target": target, "availability": availability, "requests": total, "errors": errors, "error_budget": budget, "error_budget_remaining": remaining, "within_slo": availability >= target})
}
