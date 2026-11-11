package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
)

// TenantDataHandler permanently removes tenant-owned analytics data. The
// explicit confirm=true guard prevents accidental destructive requests.
type TenantDataHandler struct{ DB *gorm.DB }

func (h *TenantDataHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Query().Get("confirm") != "true" {
		http.Error(w, "confirm=true is required", http.StatusBadRequest)
		return
	}
	tenantID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/tenants/data/"), "/")
	if tenantID == "" {
		http.Error(w, "tenant id required", http.StatusBadRequest)
		return
	}
	var tenant model.Tenant
	if err := h.DB.First(&tenant, "id = ?", tenantID).Error; err != nil {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}
	result := map[string]int64{}
	for name, query := range map[string]*gorm.DB{
		"events":    h.DB.Where("tenant_id = ?", tenantID).Delete(&model.Event{}),
		"sessions":  h.DB.Where("tenant_id = ?", tenantID).Delete(&model.Session{}),
		"schemas":   h.DB.Where("tenant_id = ?", tenantID).Delete(&model.Schema{}),
		"questions": h.DB.Where("tenant_id = ?", tenantID).Delete(&model.Question{}),
	} {
		if query.Error != nil {
			http.Error(w, query.Error.Error(), http.StatusInternalServerError)
			return
		}
		result[name] = query.RowsAffected
	}
	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"tenant_id": tenantID, "deleted": result})
}
