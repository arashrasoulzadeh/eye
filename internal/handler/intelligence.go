package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type IntelligenceHandler struct {
	Svc *service.UserIntelligenceService
}

func (h *IntelligenceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("action") == "label" && r.Method == http.MethodPost {
		var body struct {
			Fingerprint string `json:"fingerprint"`
			Label       string `json:"label"`
			Source      string `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Fingerprint == "" || body.Label == "" {
			http.Error(w, "fingerprint and label required", http.StatusBadRequest)
			return
		}
		if err := h.Svc.AddLabel(tenantID, body.Fingerprint, body.Label, body.Source); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Query().Get("action") == "readiness" {
		label := r.URL.Query().Get("label")
		if label == "" {
			label = "buyer"
		}
		result, err := h.Svc.TrainingReadiness(tenantID, label)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(result)
		return
	}
	if r.URL.Query().Get("action") == "train" && r.Method == http.MethodPost {
		label := r.URL.Query().Get("label")
		if label == "" {
			label = "buyer"
		}
		trained, err := h.Svc.Train(tenantID, label, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		json.NewEncoder(w).Encode(trained)
		return
	}
	if r.Method == http.MethodPost && r.URL.Query().Get("action") == "refresh" {
		count, err := h.Svc.Refresh(tenantID, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"refreshed": count})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := h.Svc.List(tenantID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rows)
}
