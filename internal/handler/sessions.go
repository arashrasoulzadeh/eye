package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type SessionsHandler struct{ DB *gorm.DB }

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
			StartedAt:  now,
			LastSeen:   now,
			UserAgent:  r.Header.Get("User-Agent"),
			IP:         clientIP(r),
			DeviceInfo: string(deviceJSON),
		}
		if err := h.DB.Create(s).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(s)
	case http.MethodGet:
		var list []model.Session
		h.DB.Find(&list)
		json.NewEncoder(w).Encode(list)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
