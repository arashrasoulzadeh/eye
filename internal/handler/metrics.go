package handler

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type MetricsHandler struct{ DB *gorm.DB }

func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var events int64
	h.DB.Model(&model.Event{}).Count(&events)
	var sessions int64
	h.DB.Model(&model.Session{}).Count(&sessions)
	var tenants int64
	h.DB.Model(&model.Tenant{}).Count(&tenants)
	json.NewEncoder(w).Encode(map[string]int64{"events": events, "sessions": sessions, "tenants": tenants})
}
