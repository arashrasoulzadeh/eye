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

type AdminTeamsHandler struct{ DB *gorm.DB }

type adminTeamBody struct {
	Name string `json:"name"`
}
type adminTeamMemberBody struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func adminTeamID() string { b := make([]byte, 12); _, _ = rand.Read(b); return hex.EncodeToString(b) }

func (h *AdminTeamsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/admin-teams"), "/")
	parts := strings.Split(path, "/")
	if path != "" && len(parts) >= 2 && parts[1] == "members" {
		teamID := parts[0]
		if len(parts) == 3 && r.Method == http.MethodDelete {
			if err := h.DB.Delete(&model.AdminTeamMember{}, "team_id = ? AND user_id = ?", teamID, parts[2]).Error; err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if len(parts) != 2 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			var members []model.AdminTeamMember
			if err := h.DB.Where("team_id = ?", teamID).Order("created_at desc").Find(&members).Error; err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			json.NewEncoder(w).Encode(members)
		case http.MethodPost:
			var body adminTeamMemberBody
			if json.NewDecoder(r.Body).Decode(&body) != nil || strings.TrimSpace(body.UserID) == "" {
				http.Error(w, "user_id is required", 400)
				return
			}
			var team model.AdminTeam
			var user model.AdminUser
			if h.DB.First(&team, "id = ?", teamID).Error != nil || h.DB.First(&user, "id = ?", body.UserID).Error != nil {
				http.Error(w, "team or user not found", 404)
				return
			}
			if body.Role == "" {
				body.Role = user.Role
			}
			member := model.AdminTeamMember{TeamID: teamID, UserID: body.UserID, Role: body.Role, CreatedAt: time.Now().Unix()}
			if err := h.DB.Save(&member).Error; err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(member)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if path != "" {
		if len(parts) != 1 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		teamID := parts[0]
		switch r.Method {
		case http.MethodPut:
			var body adminTeamBody
			if json.NewDecoder(r.Body).Decode(&body) != nil || strings.TrimSpace(body.Name) == "" {
				http.Error(w, "name is required", 400)
				return
			}
			if err := h.DB.Model(&model.AdminTeam{}).Where("id = ?", teamID).Updates(map[string]any{"name": strings.TrimSpace(body.Name), "updated_at": time.Now().Unix()}).Error; err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			var team model.AdminTeam
			if err := h.DB.First(&team, "id = ?", teamID).Error; err != nil {
				http.Error(w, "team not found", 404)
				return
			}
			json.NewEncoder(w).Encode(team)
		case http.MethodDelete:
			if err := h.DB.Transaction(func(tx *gorm.DB) error {
				if err := tx.Delete(&model.AdminTeamMember{}, "team_id = ?", teamID).Error; err != nil {
					return err
				}
				return tx.Delete(&model.AdminTeam{}, "id = ?", teamID).Error
			}); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	switch r.Method {
	case http.MethodGet:
		var teams []model.AdminTeam
		if err := h.DB.Order("created_at desc").Find(&teams).Error; err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(teams)
	case http.MethodPost:
		var body adminTeamBody
		if json.NewDecoder(r.Body).Decode(&body) != nil || strings.TrimSpace(body.Name) == "" {
			http.Error(w, "name is required", 400)
			return
		}
		now := time.Now().Unix()
		team := model.AdminTeam{ID: adminTeamID(), Name: strings.TrimSpace(body.Name), CreatedAt: now, UpdatedAt: now}
		if err := h.DB.Create(&team).Error; err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(team)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
