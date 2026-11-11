package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

type UsageHandler struct {
	DB  *gorm.DB
	Svc *service.UsageService
}

func (h *UsageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/tenant-usage"), "/")
	if !strings.HasSuffix(path, "/usage") {
		http.NotFound(w, r)
		return
	}
	tenantID := strings.TrimSuffix(path, "/usage")
	tenantID = strings.TrimSuffix(tenantID, "/")
	var tenant model.Tenant
	if err := h.DB.First(&tenant, "id = ?", tenantID).Error; err != nil {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}
	usage, err := h.Svc.Current(tenantID, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := struct {
		service.UsageSnapshot
		EventLimit   int64 `json:"event_limit"`
		StorageLimit int64 `json:"storage_limit"`
	}{UsageSnapshot: usage, EventLimit: tenant.MonthlyEventLimit, StorageLimit: tenant.MonthlyStorageLimit}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
