package handler

import (
	"encoding/json"
	"net/http"

	"github.com/eye/eye/internal/service"
)

type SchemasHandler struct{ Svc *service.AdminService }

func (h *SchemasHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		list, err := h.Svc.ListSchemas(tenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(list)
	case http.MethodPost:
		var b struct {
			TenantID   string `json:"tenant_id"`
			Name       string `json:"name"`
			Definition string `json:"definition"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s, err := h.Svc.RegisterSchema(b.TenantID, b.Name, b.Definition)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(s)
	default:
		w.WriteHeader(405)
	}
}
