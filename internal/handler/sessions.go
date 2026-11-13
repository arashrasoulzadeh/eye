package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

type SessionsHandler struct {
	DB               *gorm.DB
	SetTenantContext func(*gorm.DB, string) error
	GeoIP            *service.GeoIPService
}

// clientIP prefers the leftmost X-Forwarded-For entry (the original client,
// for requests behind a reverse proxy) and falls back to the raw connection
// address otherwise. This is best-effort, not a security boundary — a
// client can forge X-Forwarded-For — it's only ever used for device/session
// metadata, never for auth or rate-limit decisions.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (h *SessionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var b struct {
			TenantID string         `json:"tenant_id"`
			UserID   string         `json:"user_id"`
			Device   map[string]any `json:"device"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.TenantID == "" {
			http.Error(w, "tenant_id required", http.StatusBadRequest)
			return
		}
		idBytes := make([]byte, 16)
		_, _ = rand.Read(idBytes)
		now := time.Now()
		deviceJSON, _ := json.Marshal(b.Device)
		s := &model.Session{
			ID:         hex.EncodeToString(idBytes),
			TenantID:   b.TenantID,
			UserID:     strings.TrimSpace(b.UserID),
			StartedAt:  now,
			LastSeen:   now,
			UserAgent:  r.Header.Get("User-Agent"),
			IP:         clientIP(r),
			DeviceInfo: string(deviceJSON),
		}
		if h.GeoIP != nil {
			h.GeoIP.Enrich(s)
		}
		err := h.DB.Transaction(func(tx *gorm.DB) error {
			if h.SetTenantContext != nil {
				if err := h.SetTenantContext(tx, b.TenantID); err != nil {
					return err
				}
			}
			return tx.Create(s).Error
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(s)
	case http.MethodGet:
		if sessionID := r.URL.Query().Get("session_id"); sessionID != "" {
			var session model.Session
			if err := h.DB.First(&session, "id = ?", sessionID).Error; err != nil {
				http.Error(w, "session not found", http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(session)
			return
		}
		var list []model.Session
		limit := 100
		offset := 0
		if raw := r.URL.Query().Get("limit"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 1000 {
				limit = parsed
			}
		}
		if raw := r.URL.Query().Get("offset"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
				offset = parsed
			}
		}
		query := h.DB.Model(&model.Session{})
		if raw := r.URL.Query().Get("from_ts"); raw != "" {
			if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
				query = query.Where("last_seen >= ?", time.UnixMilli(value))
			}
		}
		if raw := r.URL.Query().Get("to_ts"); raw != "" {
			if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
				query = query.Where("last_seen <= ?", time.UnixMilli(value))
			}
		}
		if tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id")); tenantID != "" {
			query = query.Where("tenant_id = ?", tenantID)
		}
		if filter := strings.TrimSpace(r.URL.Query().Get("filter")); filter != "" {
			like := "%" + filter + "%"
			query = query.Where("id LIKE ? OR tenant_id LIKE ? OR user_agent LIKE ? OR ip LIKE ?", like, like, like, like)
		}
		var total int64
		query.Count(&total)
		order := "last_seen desc"
		if r.URL.Query().Get("sort") == "started" {
			order = "started_at desc"
		}
		if r.URL.Query().Get("dir") == "asc" {
			order = strings.Replace(order, " desc", " asc", 1)
		}
		query.Order(order).Offset(offset).Limit(limit).Find(&list)
		json.NewEncoder(w).Encode(map[string]any{"items": list, "total": total, "limit": limit, "offset": offset})
	case http.MethodPut, http.MethodPatch:
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "session_id required", http.StatusBadRequest)
			return
		}
		var b struct {
			Device    map[string]any `json:"device"`
			IP        string         `json:"ip"`
			UserAgent string         `json:"user_agent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		var s model.Session
		if err := h.DB.First(&s, "id = ?", sessionID).Error; err != nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		updates := map[string]any{"LastSeen": time.Now()}
		if b.Device != nil {
			deviceJSON, _ := json.Marshal(b.Device)
			updates["DeviceInfo"] = string(deviceJSON)
		}
		if b.IP != "" {
			updates["IP"] = b.IP
		}
		if b.UserAgent != "" {
			updates["UserAgent"] = b.UserAgent
		}
		if err := h.DB.Model(&s).Updates(updates).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.DB.First(&s, "id = ?", sessionID)
		json.NewEncoder(w).Encode(s)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
