package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
	"gorm.io/gorm"
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
	if r.URL.Query().Get("action") == "suggestions" {
		rows, err := h.Svc.LabelSuggestions(tenantID, 50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(rows)
		return
	}
	if r.URL.Query().Get("action") == "review" && r.Method == http.MethodPost {
		var body struct {
			Fingerprint string `json:"fingerprint"`
			Label       string `json:"label"`
			Decision    string `json:"decision"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Fingerprint == "" || body.Label == "" || body.Decision == "" {
			http.Error(w, "invalid review", http.StatusBadRequest)
			return
		}
		if err := h.Svc.ReviewSuggestion(tenantID, body.Fingerprint, body.Label, body.Decision); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "fingerprint": body.Fingerprint, "label": body.Label, "decision": body.Decision})
		return
	}
	if r.URL.Query().Get("action") == "pipeline" && r.Method == http.MethodPost {
		retryID := strings.TrimSpace(r.URL.Query().Get("retry_id"))
		var run model.IntelligencePipelineRun
		var err error
		if retryID != "" {
			run, err = h.Svc.RetryPipeline(tenantID, retryID, time.Now())
		} else {
			run, err = h.Svc.StartPipeline(tenantID, time.Now())
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(run)
		return
	}
	if r.URL.Query().Get("action") == "pipeline-status" && r.Method == http.MethodGet {
		runs, err := h.Svc.PipelineRuns(tenantID, 20)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(runs)
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
	if r.URL.Query().Get("action") == "models" && r.Method == http.MethodGet {
		models, err := h.Svc.Models(tenantID, r.URL.Query().Get("label"), 100)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(models)
		return
	}
	if r.URL.Query().Get("action") == "model-layers" && r.Method == http.MethodGet {
		models, err := h.Svc.ModelLayers(tenantID, r.URL.Query().Get("label"), 100)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(models)
		return
	}
	if r.URL.Query().Get("action") == "evaluations" && r.Method == http.MethodGet {
		evaluations, err := h.Svc.Evaluations(tenantID, r.URL.Query().Get("label"), 100)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(evaluations)
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
			if evaluation, err := h.Svc.EvaluateLatest("", "behavior", time.Now()); err != nil {
				http.Error(w, "global model evaluation failed: "+err.Error(), http.StatusUnprocessableEntity)
				return
			} else if evaluation.Status == "drifted" {
				http.Error(w, "global model is drifted and cannot be promoted", http.StatusUnprocessableEntity)
				return
			} else if _, err := h.Svc.PromoteLatest("", "behavior", time.Now()); err != nil {
				http.Error(w, "global model promotion failed: "+err.Error(), http.StatusUnprocessableEntity)
				return
			}
			trained, err := h.Svc.TrainUnsupervised(tenantID, time.Now())
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			if evaluation, err := h.Svc.EvaluateLatest(tenantID, "behavior", time.Now()); err != nil {
				http.Error(w, "tenant model evaluation failed: "+err.Error(), http.StatusUnprocessableEntity)
				return
			} else if evaluation.Status == "drifted" {
				http.Error(w, "tenant model is drifted and cannot be promoted", http.StatusUnprocessableEntity)
				return
			} else if _, err := h.Svc.PromoteLatest(tenantID, "behavior", time.Now()); err != nil {
				http.Error(w, "tenant model promotion failed: "+err.Error(), http.StatusUnprocessableEntity)
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
	if r.URL.Query().Get("action") == "evaluation" {
		label := r.URL.Query().Get("label")
		if label == "" {
			label = "behavior"
		}
		if r.Method == http.MethodPost {
			result, err := h.Svc.EvaluateLatest(tenantID, label, time.Now())
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			if err := h.Svc.RecordQualityAlert(result); err != nil {
				http.Error(w, "evaluation recorded but quality alert persistence failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(result)
			return
		}
		result, err := h.Svc.LatestEvaluation(tenantID, label)
		if err != nil {
			http.Error(w, "evaluation unavailable", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(result)
		return
	}
	if r.URL.Query().Get("action") == "quality" {
		if r.Method == http.MethodPost {
			result, err := h.Svc.AssessDataQuality(tenantID, time.Now())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(result)
			return
		}
		result, err := h.Svc.LatestDataQuality(tenantID)
		if err != nil {
			http.Error(w, "data quality unavailable", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(result)
		return
	}
	if r.URL.Query().Get("action") == "alerts" && r.Method == http.MethodGet {
		alerts, err := h.Svc.IntelligenceAlerts(tenantID, r.URL.Query().Get("status"), 100)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(alerts)
		return
	}
	if r.URL.Query().Get("action") == "resolve-alert" && r.Method == http.MethodPost {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "alert id required", http.StatusBadRequest)
			return
		}
		if err := h.Svc.ResolveIntelligenceAlert(tenantID, id, time.Now()); err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "open alert not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Query().Get("action") == "promote" && r.Method == http.MethodPost {
		label := r.URL.Query().Get("label")
		if label == "" {
			label = "behavior"
		}
		result, err := h.Svc.PromoteLatest(tenantID, label, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		json.NewEncoder(w).Encode(result)
		return
	}
	if r.URL.Query().Get("action") == "rollback" && r.Method == http.MethodPost {
		label := r.URL.Query().Get("label")
		if label == "" {
			label = "behavior"
		}
		result, err := h.Svc.Rollback(tenantID, label)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		json.NewEncoder(w).Encode(result)
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
