package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type TenantsHandler struct{ Svc *service.AdminService }

func (h *TenantsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/tenants")
	path = strings.Trim(path, "/")

	if r.Method == http.MethodPost && strings.HasSuffix(path, "/rotate-key") {
		tenantID := strings.TrimSuffix(path, "/rotate-key")
		newKey, err := h.Svc.RotateKey(tenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"api_key": newKey})
		return
	}

	if r.Method == http.MethodPost && strings.HasSuffix(path, "/revoke-key") {
		tenantID := strings.TrimSuffix(path, "/revoke-key")
		if err := h.Svc.RevokeKey(tenantID); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method == http.MethodPut && strings.HasSuffix(path, "/tier") {
		tenantID := strings.TrimSuffix(path, "/tier")
		var b struct {
			Tier string `json:"tier"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.Svc.SetTier(tenantID, b.Tier); err != nil {
			status := http.StatusBadRequest
			if err == gorm.ErrRecordNotFound {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodPut && strings.HasSuffix(path, "/quota") {
		tenantID := strings.TrimSuffix(path, "/quota")
		var b struct {
			EventLimit   int64 `json:"event_limit"`
			StorageLimit int64 `json:"storage_limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.Svc.SetQuotas(tenantID, b.EventLimit, b.StorageLimit); err != nil {
			status := http.StatusBadRequest
			if err == gorm.ErrRecordNotFound {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case http.MethodGet:
		list, _ := h.Svc.ListTenants()
		json.NewEncoder(w).Encode(list)
	case http.MethodPost:
		var b struct{ Name, APIKey string }
		json.NewDecoder(r.Body).Decode(&b)
		t, _ := h.Svc.CreateTenant(b.Name, b.APIKey)
		json.NewEncoder(w).Encode(t)
	case http.MethodDelete:
		if path == "" {
			http.Error(w, "tenant id required", http.StatusBadRequest)
			return
		}
		if err := h.Svc.DeleteTenant(path); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(405)
	}
}
