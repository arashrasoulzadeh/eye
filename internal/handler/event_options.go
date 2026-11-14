package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type EventOption struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
	Last  int64  `json:"last_ts"`
}

// EventOptionsHandler returns searchable event-name cardinalities. Defaults
// are the most frequent names, with latest activity breaking ties.
type EventOptionsHandler struct{ DB *gorm.DB }

func (h *EventOptionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenant, query := r.URL.Query().Get("tenant_id"), r.URL.Query().Get("q")
	all := r.URL.Query().Get("all") == "1" || r.URL.Query().Get("all") == "true"
	var rows []EventOption
	db := h.DB.Table("events").Select("event_name as value, COUNT(*) as count, MAX(client_ts) as last").Where("event_name <> ''")
	if tenant != "" {
		db = db.Where("tenant_id = ?", tenant)
	}
	if from, err := strconv.ParseInt(r.URL.Query().Get("from_ts"), 10, 64); err == nil {
		db = db.Where("client_ts >= ?", from)
	}
	if to, err := strconv.ParseInt(r.URL.Query().Get("to_ts"), 10, 64); err == nil {
		db = db.Where("client_ts <= ?", to)
	}
	if query != "" {
		db = db.Where("event_name LIKE ?", "%"+strings.ReplaceAll(query, "%", "\\%")+"%")
	}
	if !all {
		db = db.Limit(5)
	}
	if err := db.Group("event_name").Order("count DESC, last DESC").Scan(&rows).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(rows)
}
