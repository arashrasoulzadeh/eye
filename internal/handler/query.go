package handler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type QueryHandler struct{ Svc *service.QueryService }
type queryCursor struct {
	ClientTS int64  `json:"client_ts"`
	EventID  string `json:"event_id"`
}

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
		SortDesc:     q.Get("sort") == "desc",
	}
	if raw := q.Get("cursor"); raw != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		var cursor queryCursor
		if err != nil || json.Unmarshal(decoded, &cursor) != nil || cursor.EventID == "" {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		f.AfterClientTS = &cursor.ClientTS
		f.AfterEventID = cursor.EventID
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
	if len(list) == limit {
		last := list[len(list)-1]
		encoded, _ := json.Marshal(queryCursor{ClientTS: last.ClientTS, EventID: last.EventID})
		w.Header().Set("X-Next-Cursor", base64.RawURLEncoding.EncodeToString(encoded))
	}
	json.NewEncoder(w).Encode(list)
}
