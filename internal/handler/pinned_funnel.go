package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PinnedFunnelHandler struct{ DB *gorm.DB }

// GET returns the definition or null. POST replaces it. DELETE clears it.
func (h *PinnedFunnelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	db := h.DB.WithContext(r.Context())
	var funnel model.PinnedFunnel
	switch r.Method {
	case http.MethodGet:
		err := db.First(&funnel, "tenant_id = ?", tenantID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writePinnedFunnel(w, nil)
			return
		}
		if err != nil {
			http.Error(w, "could not load pinned funnel", http.StatusInternalServerError)
			return
		}
		writePinnedFunnel(w, &funnel)
	case http.MethodPost:
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&funnel); err != nil {
			http.Error(w, "invalid funnel definition", http.StatusBadRequest)
			return
		}
		funnel.Name = strings.TrimSpace(funnel.Name)
		if funnel.Name == "" || len(funnel.Steps) < 2 {
			http.Error(w, "a name and at least two steps are required", http.StatusBadRequest)
			return
		}
		seen := make(map[string]bool, len(funnel.Steps))
		for index, step := range funnel.Steps {
			step = strings.TrimSpace(step)
			if step == "" || seen[step] {
				http.Error(w, "steps must be non-empty and distinct", http.StatusBadRequest)
				return
			}
			seen[step] = true
			funnel.Steps[index] = step
		}
		var tenant model.Tenant
		if err := db.Select("id").First(&tenant, "id = ?", tenantID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "tenant not found", http.StatusNotFound)
			} else {
				http.Error(w, "could not look up tenant", http.StatusInternalServerError)
			}
			return
		}
		funnel.TenantID = tenantID
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "steps"}),
		}).Create(&funnel).Error
		if err != nil {
			http.Error(w, "could not save pinned funnel", http.StatusInternalServerError)
			return
		}
		writePinnedFunnel(w, &funnel)
	case http.MethodDelete:
		if err := db.Delete(&funnel, "tenant_id = ?", tenantID).Error; err != nil {
			http.Error(w, "could not clear pinned funnel", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func writePinnedFunnel(w http.ResponseWriter, funnel *model.PinnedFunnel) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(funnel)
}
