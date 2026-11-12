package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
)

type AdminInvitationHandler struct{ DB *gorm.DB }

func (h *AdminInvitationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil || strings.TrimSpace(body.Token) == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}
	hash := sha256.Sum256([]byte(strings.TrimSpace(body.Token)))
	var user model.AdminUser
	if err := h.DB.Where("invite_token_hash = ? AND invite_expires_at > ? AND status = ?", hex.EncodeToString(hash[:]), time.Now().Unix(), "invited").First(&user).Error; err != nil {
		http.Error(w, "invitation is invalid or expired", http.StatusUnauthorized)
		return
	}
	user.Status = "active"
	user.InviteTokenHash = ""
	user.InviteExpiresAt = 0
	user.UpdatedAt = time.Now().Unix()
	if err := h.DB.Save(&user).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(user)
}
