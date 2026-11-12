package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
)

func adminSessionToken() (string, string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	hash := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(hash[:])
}

func adminSessionID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// AdminSessionAuth accepts a durable revocable session, with the configured
// shared key retained as a migration fallback for existing deployments.
func AdminSessionAuth(db *gorm.DB, legacyKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secureKeyEqual(legacyKey, r.Header.Get("x-admin-key")) {
			next.ServeHTTP(w, r)
			return
		}
		token := strings.TrimSpace(r.Header.Get("x-admin-session"))
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		hash := sha256.Sum256([]byte(token))
		var session model.AdminSession
		if err := db.Where("token_hash = ? AND revoked_at = 0 AND expires_at > ?", hex.EncodeToString(hash[:]), time.Now().Unix()).First(&session).Error; err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if session.Email != "" {
			r.Header.Set("x-admin-email", session.Email)
		}
		next.ServeHTTP(w, r)
	})
}

func RevokeAdminSession(db *gorm.DB, token string) error {
	hash := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return db.Model(&model.AdminSession{}).Where("token_hash = ? AND revoked_at = 0", hex.EncodeToString(hash[:])).Update("revoked_at", time.Now().Unix()).Error
}

type AdminSessionsHandler struct {
	DB       *gorm.DB
	AdminKey string
}

func AdminSessionRoleAuth(db *gorm.DB, adminKey string, resolveRole func(string) (string, bool), allowed []string, next http.Handler) http.Handler {
	return AdminSessionAuth(db, adminKey, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email := r.Header.Get("x-admin-email")
		if email != "" {
			role, ok := resolveRole(email)
			if !ok || !containsRole(allowed, role) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	}))
}

func containsRole(allowed []string, role string) bool {
	for _, value := range allowed {
		if value == role {
			return true
		}
	}
	return false
}

func (h *AdminSessionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		if err := RevokeAdminSession(h.DB, r.Header.Get("x-admin-session")); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost || !secureKeyEqual(h.AdminKey, r.Header.Get("x-admin-key")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var body struct {
		Email  string `json:"email"`
		UserID string `json:"user_id"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	if body.Email != "" || body.UserID != "" {
		var user model.AdminUser
		query := h.DB.Where("status = ?", "active")
		if body.UserID != "" {
			query = query.Where("id = ?", body.UserID)
		} else {
			query = query.Where("email = ?", strings.TrimSpace(body.Email))
		}
		if err := query.First(&user).Error; err != nil {
			http.Error(w, "active admin user not found", http.StatusForbidden)
			return
		}
		body.Email, body.UserID = user.Email, user.ID
	}
	token, hash := adminSessionToken()
	now := time.Now().Unix()
	session := model.AdminSession{ID: adminSessionID(), TokenHash: hash, UserID: body.UserID, Email: strings.TrimSpace(body.Email), CreatedAt: now, ExpiresAt: now + 8*60*60}
	if err := h.DB.Create(&session).Error; err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"token": token, "expires_at": session.ExpiresAt, "email": session.Email})
}
