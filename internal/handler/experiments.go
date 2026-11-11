package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type ExperimentsHandler struct{ Svc *service.ExperimentService }

func (h *ExperimentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/experiments")
	path = strings.Trim(path, "/")

	if r.Method == http.MethodGet && strings.HasSuffix(path, "/assign") {
		experimentID := strings.TrimSuffix(path, "/assign")
		experimentID = strings.TrimSuffix(experimentID, "/")
		sessionID := r.URL.Query().Get("session_id")
		if experimentID == "" || sessionID == "" {
			http.Error(w, "experiment id and session_id are required", http.StatusBadRequest)
			return
		}
		variant, err := h.Svc.AssignVariant(experimentID, sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"experiment_id": experimentID, "variant": variant})
		return
	}
	if r.Method == http.MethodGet && strings.HasSuffix(path, "/significance") {
		q := r.URL.Query()
		var controlSuccesses, controlSamples, variantSuccesses, variantSamples int64
		if _, err := fmt.Sscan(q.Get("control_successes"), &controlSuccesses); err != nil {
			http.Error(w, "invalid control_successes", http.StatusBadRequest)
			return
		}
		if _, err := fmt.Sscan(q.Get("control_samples"), &controlSamples); err != nil {
			http.Error(w, "invalid control_samples", http.StatusBadRequest)
			return
		}
		if _, err := fmt.Sscan(q.Get("variant_successes"), &variantSuccesses); err != nil {
			http.Error(w, "invalid variant_successes", http.StatusBadRequest)
			return
		}
		if _, err := fmt.Sscan(q.Get("variant_samples"), &variantSamples); err != nil {
			http.Error(w, "invalid variant_samples", http.StatusBadRequest)
			return
		}
		result, err := service.Significance(controlSuccesses, controlSamples, variantSuccesses, variantSamples)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(result)
		return
	}
	if r.Method == http.MethodPost && strings.HasSuffix(path, "/auto-stop") {
		experimentID := strings.TrimSuffix(strings.TrimSuffix(path, "/auto-stop"), "/")
		var counts struct {
			ControlSuccesses int64 `json:"control_successes"`
			ControlSamples   int64 `json:"control_samples"`
			VariantSuccesses int64 `json:"variant_successes"`
			VariantSamples   int64 `json:"variant_samples"`
		}
		if err := json.NewDecoder(r.Body).Decode(&counts); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := service.Significance(counts.ControlSuccesses, counts.ControlSamples, counts.VariantSuccesses, counts.VariantSamples)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		stopped, err := h.Svc.AutoStopIfWinner(experimentID, result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"stopped": stopped, "significance": result})
		return
	}

	switch r.Method {
	case http.MethodGet:
		list, err := h.Svc.List(r.URL.Query().Get("tenant_id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(list)
	case http.MethodPost:
		var b struct {
			Name     string
			TenantID string
			Variants string
			Traffic  int
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		exp, err := h.Svc.Create(b.Name, b.TenantID, b.Variants, b.Traffic)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(exp)
	case http.MethodPut:
		id := strings.Trim(path, "/")
		var b struct {
			Name     string `json:"name"`
			Variants string `json:"variants"`
			Traffic  int    `json:"traffic"`
			Enabled  bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		exp, err := h.Svc.Update(id, b.Name, b.Variants, b.Traffic, b.Enabled)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(exp)
	case http.MethodDelete:
		if err := h.Svc.Delete(strings.Trim(path, "/")); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
