package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

type EventsHandler struct{ DB *gorm.DB }

func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/events"), "/")
	if r.Method == http.MethodGet && path != "" {
		var event model.Event
		if err := h.DB.First(&event, "event_id = ?", path).Error; err != nil {
			http.Error(w, "event not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(event)
		return
	}
	var list []model.Event
	q := h.DB
	if n := r.URL.Query().Get("event_name"); n != "" {
		q = q.Where("event_name = ?", n)
	}
	if t := r.URL.Query().Get("tenant_id"); t != "" {
		q = q.Where("tenant_id = ?", t)
	}
	if s := r.URL.Query().Get("session_id"); s != "" {
		q = q.Where("session_id = ?", s)
	}
	q.Order("ingest_ts desc").Limit(100).Find(&list)
	json.NewEncoder(w).Encode(list)
}
