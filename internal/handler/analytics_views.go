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

type AnalyticsViewsHandler struct{ DB *gorm.DB }

func (h *AnalyticsViewsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/analytics/views"), "/")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	if r.Method == http.MethodGet && path == "" {
		var views []model.AnalyticsView
		if err := h.DB.Where("tenant_id = ?", tenantID).Order("updated_at desc").Find(&views).Error; err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(views)
		return
	}
	if r.Method == http.MethodDelete && path != "" {
		if err := h.DB.Where("id = ? AND tenant_id = ?", path, tenantID).Delete(&model.AnalyticsView{}).Error; err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if (r.Method != http.MethodPost && r.Method != http.MethodPut) || (r.Method == http.MethodPut && path == "") {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var view model.AnalyticsView
	if err := json.NewDecoder(r.Body).Decode(&view); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	view.TenantID = tenantID
	if r.Method == http.MethodPut {
		view.ID = path
	}
	if view.ID == "" {
		raw := make([]byte, 16)
		if _, err := rand.Read(raw); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		view.ID = hex.EncodeToString(raw)
		view.CreatedAt = time.Now().Unix()
	}
	view.UpdatedAt = time.Now().Unix()
	if err := h.DB.Save(&view).Error; err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(view)
}
