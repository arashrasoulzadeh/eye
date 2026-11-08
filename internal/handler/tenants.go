package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/eye/eye/internal/service"
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

	switch r.Method {
	case http.MethodGet:
		list, _ := h.Svc.ListTenants()
		json.NewEncoder(w).Encode(list)
	case http.MethodPost:
		var b struct{ Name, APIKey string }
		json.NewDecoder(r.Body).Decode(&b)
		t, _ := h.Svc.CreateTenant(b.Name, b.APIKey)
		json.NewEncoder(w).Encode(t)
	default:
		w.WriteHeader(405)
	}
}
