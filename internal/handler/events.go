package handler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

type EventsHandler struct{ DB *gorm.DB }

type eventCursor struct {
	IngestTS time.Time `json:"ingest_ts"`
	EventID  string    `json:"event_id"`
}

func decodeEventCursor(raw string) (eventCursor, error) {
	var c eventCursor
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}

func encodeEventCursor(c eventCursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

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
	limit := 100
	offset := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			offset = n
		}
	}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		cursor, err := decodeEventCursor(raw)
		if err != nil || cursor.EventID == "" {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		q = q.Where("(ingest_ts < ?) OR (ingest_ts = ? AND event_id < ?)", cursor.IngestTS, cursor.IngestTS, cursor.EventID)
	}
	direction := "desc"
	if r.URL.Query().Get("sort") == "asc" {
		direction = "asc"
	}
	q.Order("ingest_ts " + direction + ", event_id " + direction).Offset(offset).Limit(limit + 1).Find(&list)
	if len(list) > limit {
		last := list[limit-1]
		list = list[:limit]
		w.Header().Set("X-Next-Cursor", encodeEventCursor(eventCursor{IngestTS: last.IngestTS, EventID: last.EventID}))
	}
	json.NewEncoder(w).Encode(list)
}
