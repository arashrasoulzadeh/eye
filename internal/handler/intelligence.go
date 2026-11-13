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
	Svc      *service.UserIntelligenceService
	Registry service.ModelRegistry
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
	if r.URL.Query().Get("action") == "training" {
		runs, err := h.Svc.TrainingRuns(tenantID, r.URL.Query().Get("label"), 20)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(runs)
		return
	}
	if r.URL.Query().Get("action") == "train" && r.Method == http.MethodPost {
		label := r.URL.Query().Get("label")
		if label == "" {
			label = "auto"
		}
		if label == "auto" {
			if _, err := h.Svc.TrainUnsupervisedGlobal(time.Now()); err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			trained, err := h.Svc.TrainUnsupervised(tenantID, time.Now())
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			json.NewEncoder(w).Encode(trained)
			return
		}
		trained, err := h.Svc.Train(tenantID, label, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		json.NewEncoder(w).Encode(trained)
		return
	}
	if r.URL.Query().Get("action") == "model" {
		task := r.URL.Query().Get("task")
		if task == "" {
			task = "buyer"
		}
		manifest, scope, err := h.Registry.Resolve(tenantID, task)
		if err != nil {
			http.Error(w, "offline model unavailable", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"scope": scope, "manifest": manifest})
		return
	}
	if r.URL.Query().Get("action") == "export" && r.Method == http.MethodPost {
		label := r.URL.Query().Get("label")
		if label == "" {
			label = "buyer"
		}
		scope := r.URL.Query().Get("scope")
		if scope == "" {
			scope = "global"
		}
		if scope != "global" {
			scope = "tenants/" + tenantID
		}
		manifest, err := h.Svc.ExportModel(tenantID, label, h.Registry, scope, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		json.NewEncoder(w).Encode(manifest)
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
