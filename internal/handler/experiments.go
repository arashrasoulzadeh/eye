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

	switch r.Method {
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
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
