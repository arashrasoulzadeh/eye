package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/eye/eye/internal/service"
)

type QueryHandler struct{ Svc *service.QueryService }

func (h *QueryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 100
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	offset := 0
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	f := service.EventFilter{
		TenantID:     q.Get("tenant_id"),
		SessionID:    q.Get("session_id"),
		EventName:    q.Get("event_name"),
		DimsContains: q.Get("dims_contains"),
	}
	if v := q.Get("from_ts"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.FromTS = &n
		}
	}
	if v := q.Get("to_ts"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.ToTS = &n
		}
	}

	list, err := h.Svc.Filter(f, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(list)
}
