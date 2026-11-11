package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
)

type AdminUsersHandler struct{ DB *gorm.DB }

type adminUserBody struct {
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

func adminUserID() string { b := make([]byte, 12); _, _ = rand.Read(b); return hex.EncodeToString(b) }

func (h *AdminUsersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/admin-users"), "/")
	if id != "" {
		if r.Method == http.MethodDelete {
			if err := h.DB.Delete(&model.AdminUser{}, "id = ?", id).Error; err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body adminUserBody
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			http.Error(w, "invalid body", 400)
			return
		}
		var user model.AdminUser
		if err := h.DB.First(&user, "id = ?", id).Error; err != nil {
			http.Error(w, "not found", 404)
			return
		}
		user.Email = strings.TrimSpace(body.Email)
		user.Name = strings.TrimSpace(body.Name)
		user.Role = body.Role
		user.Status = body.Status
		user.UpdatedAt = time.Now().Unix()
		if err := h.DB.Save(&user).Error; err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		json.NewEncoder(w).Encode(user)
		return
	}
	switch r.Method {
	case http.MethodGet:
		var users []model.AdminUser
		q := h.DB.Order("created_at desc")
		if filter := strings.TrimSpace(r.URL.Query().Get("filter")); filter != "" {
			like := "%" + filter + "%"
			q = q.Where("email LIKE ? OR name LIKE ? OR role LIKE ?", like, like, like)
		}
		if err := q.Find(&users).Error; err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(users)
	case http.MethodPost:
		var body adminUserBody
		if json.NewDecoder(r.Body).Decode(&body) != nil || strings.TrimSpace(body.Email) == "" {
			http.Error(w, "email is required", 400)
			return
		}
		if body.Role == "" {
			body.Role = "viewer"
		}
		if body.Status == "" {
			body.Status = "invited"
		}
		now := time.Now().Unix()
		user := model.AdminUser{ID: adminUserID(), Email: strings.TrimSpace(body.Email), Name: strings.TrimSpace(body.Name), Role: body.Role, Status: body.Status, CreatedAt: now, UpdatedAt: now}
		if err := h.DB.Create(&user).Error; err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
