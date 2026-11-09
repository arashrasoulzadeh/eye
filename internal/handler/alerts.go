package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/eye/eye/internal/service"
)

type AlertsHandler struct{ Svc *service.AlertService }

type alertRuleBody struct {
	TenantID   string `json:"tenant_id"`
	EventName  string `json:"event_name"`
	DimsKey    string `json:"dims_key"`
	Metric     string `json:"metric"`
	Threshold  int64  `json:"threshold"`
	WebhookURL string `json:"webhook_url"`
}

func (h *AlertsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/alerts"), "/")

	if path != "" {
		switch r.Method {
		case http.MethodGet:
			rule, err := h.Svc.Get(path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(rule)
		case http.MethodPut:
			var b alertRuleBody
			if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			rule, err := h.Svc.Update(path, b.EventName, b.DimsKey, b.Metric, b.Threshold, b.WebhookURL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(rule)
		case http.MethodDelete:
			if err := h.Svc.Delete(path); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
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
		var b alertRuleBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rule, err := h.Svc.Create(b.TenantID, b.EventName, b.DimsKey, b.Metric, b.Threshold, b.WebhookURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(rule)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
