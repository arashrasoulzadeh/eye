package handler

import (
	"encoding/json"
	"net/http"
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
