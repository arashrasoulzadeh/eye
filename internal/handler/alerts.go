package handler

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type AlertsHandler struct{ Svc *service.AlertService }

type alertRuleBody struct {
	TenantID      string `json:"tenant_id"`
	EventName     string `json:"event_name"`
	DimsKey       string `json:"dims_key"`
	Metric        string `json:"metric"`
	Threshold     int64  `json:"threshold"`
	WindowSeconds int64  `json:"window_seconds"`
	WebhookURL    string `json:"webhook_url"`
}

func (h *AlertsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/alerts"), "/")

	if r.Method == http.MethodPost && (strings.HasSuffix(path, "/enable") || strings.HasSuffix(path, "/disable")) {
		enabled := strings.HasSuffix(path, "/enable")
		id := strings.TrimSuffix(strings.TrimSuffix(path, "/enable"), "/disable")
		rule, err := h.Svc.SetEnabled(id, enabled)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(rule)
		return
	}

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
			rule, err := h.Svc.UpdateWithWindow(path, b.EventName, b.DimsKey, b.Metric, b.Threshold, b.WindowSeconds, b.WebhookURL)
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
		filter := strings.ToLower(r.URL.Query().Get("filter"))
		if filter != "" {
			filtered := list[:0]
			for _, item := range list {
				if strings.Contains(strings.ToLower(item.EventName), filter) || strings.Contains(strings.ToLower(item.Metric), filter) {
					filtered = append(filtered, item)
				}
			}
			list = filtered
		}
		sort.SliceStable(list, func(i, j int) bool {
			if r.URL.Query().Get("dir") == "asc" {
				return list[i].CreatedAt < list[j].CreatedAt
			}
			return list[i].CreatedAt > list[j].CreatedAt
		})
		if r.URL.Query().Get("page") == "true" {
			total := len(list)
			limit := queryInt(r, "limit", 10)
			offset := queryInt(r, "offset", 0)
			if limit < 1 {
				limit = 10
			}
			if offset < 0 {
				offset = 0
			}
			if offset > total {
				offset = total
			}
			end := offset + limit
			if end > total {
				end = total
			}
			json.NewEncoder(w).Encode(map[string]any{"items": list[offset:end], "total": total, "limit": limit, "offset": offset})
			return
		}
		json.NewEncoder(w).Encode(list)
	case http.MethodPost:
		var b alertRuleBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rule, err := h.Svc.CreateWithWindow(b.TenantID, b.EventName, b.DimsKey, b.Metric, b.Threshold, b.WindowSeconds, b.WebhookURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(rule)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
