package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type AggregationsHandler struct{ Svc *service.AggregatorService }

type AggregationExportHandler struct {
	Svc    *service.AggregatorService
	Client service.WebhookClient
}

func (h *AggregationExportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		TenantID  string `json:"tenant_id"`
		EventName string `json:"event_name"`
		URL       string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Svc.ExportAggregateCounts(body.TenantID, body.EventName, body.URL, h.Client); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"exported": true})
}

type aggregationBody struct {
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name"`
	Rule        string `json:"rule"`
	OutputEvent string `json:"output_event"`
	Threshold   int    `json:"threshold"`
	Window      string `json:"window"`
}

func (h *AggregationsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/aggregations"), "/")
	if path == "refresh" && r.Method == http.MethodPost {
		tenantID := r.URL.Query().Get("tenant_id")
		if err := h.Svc.RefreshMaterializedCounts(tenantID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"refreshed": true})
		return
	}

	if r.Method == http.MethodPost && strings.HasSuffix(path, "/backfill") {
		id := strings.TrimSuffix(path, "/backfill")
		n, err := h.Svc.Backfill(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]int{"emitted": n})
		return
	}

	if path != "" {
		switch r.Method {
		case http.MethodGet:
			agg, err := h.Svc.Get(path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(agg)
		case http.MethodPut:
			var b aggregationBody
			if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			agg, err := h.Svc.Update(path, b.Name, b.Rule, b.OutputEvent, b.Threshold, b.Window)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(agg)
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
		if r.URL.Query().Get("page") == "true" {
			limit, offset := queryInt(r, "limit", 10), queryInt(r, "offset", 0)
			list, total, err := h.Svc.ListPage(r.URL.Query().Get("tenant_id"), r.URL.Query().Get("filter"), r.URL.Query().Get("dir"), limit, offset)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"items": list, "total": total, "limit": limit, "offset": offset})
			return
		}
		list, err := h.Svc.List(r.URL.Query().Get("tenant_id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(list)
	case http.MethodPost:
		var b aggregationBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		agg, err := h.Svc.Create(b.Name, b.TenantID, b.Rule, b.OutputEvent, b.Threshold, b.Window)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(agg)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
