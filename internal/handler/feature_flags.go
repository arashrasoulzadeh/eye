package handler

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/service"
)

type FeatureFlagsHandler struct{ Svc *service.FeatureFlagService }

func (h *FeatureFlagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		list, err := h.Svc.List(tenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(list)
	case http.MethodPost, http.MethodPut:
		var b struct {
			TenantID string `json:"tenant_id"`
			Key      string `json:"key"`
			Enabled  bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.TenantID == "" || b.Key == "" {
			http.Error(w, "tenant_id and key required", http.StatusBadRequest)
			return
		}
		f, err := h.Svc.Set(b.TenantID, b.Key, b.Enabled)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(f)
	case http.MethodDelete:
		tenantID := r.URL.Query().Get("tenant_id")
		key := r.URL.Query().Get("key")
		if tenantID == "" || key == "" {
			http.Error(w, "tenant_id and key required", http.StatusBadRequest)
			return
		}
		if err := h.Svc.Delete(tenantID, key); err != nil {
			status := http.StatusInternalServerError
			if err == gorm.ErrRecordNotFound {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
