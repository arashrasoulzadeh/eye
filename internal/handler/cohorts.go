package handler

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type CohortsHandler struct{ Svc *service.CohortService }

func (h *CohortsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/cohorts"), "/")

	if r.Method == http.MethodPost && strings.HasSuffix(path, "/materialize") {
		id := strings.TrimSuffix(path, "/materialize")
		n, err := h.Svc.Materialize(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]int{"members": n})
		return
	}
	if r.Method == http.MethodGet && strings.HasSuffix(path, "/members") {
		id := strings.TrimSuffix(path, "/members")
		members, err := h.Svc.ListMembers(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(members)
		return
	}
	if path != "" && r.Method == http.MethodDelete {
		if err := h.Svc.Delete(path); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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
				if strings.Contains(strings.ToLower(item.Name), filter) || strings.Contains(strings.ToLower(item.EventName), filter) {
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
		var b struct {
			TenantID  string `json:"tenant_id"`
			Name      string `json:"name"`
			EventName string `json:"event_name"`
			MinCount  int    `json:"min_count"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		c, err := h.Svc.Create(b.TenantID, b.Name, b.EventName, b.MinCount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(c)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
