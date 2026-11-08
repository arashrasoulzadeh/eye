package handler

import (
	"encoding/json"
	"net/http"

	"github.com/eye/eye/internal/service"
)

type TenantsHandler struct{ Svc *service.AdminService }

func (h *TenantsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
