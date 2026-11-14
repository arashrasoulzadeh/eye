package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// AuditLogHandler serves GET /audit-log, paginated, newest first.
type AuditLogHandler struct{ DB *gorm.DB }

func (h *AuditLogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	query := h.DB.Model(&model.AuditLog{})
	if from, err := strconv.ParseInt(r.URL.Query().Get("from_ts"), 10, 64); err == nil {
		query = query.Where("created_at >= ?", (from+999)/1000)
	}
	if to, err := strconv.ParseInt(r.URL.Query().Get("to_ts"), 10, 64); err == nil {
		query = query.Where("created_at <= ?", to/1000)
	}
	if filter := r.URL.Query().Get("filter"); filter != "" {
		like := "%" + filter + "%"
		query = query.Where("action LIKE ? OR resource LIKE ? OR actor_key LIKE ? OR body_snippet LIKE ?", like, like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	order := "created_at desc"
	if r.URL.Query().Get("dir") == "asc" {
		order = "created_at asc"
	}
	var list []model.AuditLog
	if err := query.Order(order).Limit(limit).Offset(offset).Find(&list).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.URL.Query().Get("page") == "true" {
		json.NewEncoder(w).Encode(map[string]any{"items": list, "total": total, "limit": limit, "offset": offset})
		return
	}
	json.NewEncoder(w).Encode(list)
}
