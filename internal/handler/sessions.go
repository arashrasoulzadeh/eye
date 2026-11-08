package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type SessionsHandler struct{ DB *gorm.DB }

func (h *SessionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var b struct {
			TenantID string `json:"tenant_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.TenantID == "" {
			http.Error(w, "tenant_id required", http.StatusBadRequest)
			return
		}
		idBytes := make([]byte, 16)
		_, _ = rand.Read(idBytes)
		now := time.Now()
		s := &model.Session{
			ID:        hex.EncodeToString(idBytes),
			TenantID:  b.TenantID,
			StartedAt: now,
			LastSeen:  now,
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
