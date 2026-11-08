package handler

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type EventsHandler struct{ DB *gorm.DB }

func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var list []model.Event
	q := h.DB
	if n := r.URL.Query().Get("event_name"); n != "" {
		q = q.Where("event_name = ?", n)
	}
	q.Limit(100).Find(&list)
	json.NewEncoder(w).Encode(list)
}
