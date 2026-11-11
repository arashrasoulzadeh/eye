package handler

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type SchemasHandler struct{ Svc *service.AdminService }

func (h *SchemasHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/schemas")
	path = strings.Trim(path, "/")

	if r.Method == http.MethodPost && strings.HasSuffix(path, "/validate") {
		schemaID := strings.TrimSuffix(path, "/validate")
		var b struct {
			Payload map[string]any `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		schema, err := h.Svc.GetSchema(schemaID)
		if err != nil {
			http.Error(w, "schema not found", http.StatusNotFound)
			return
		}
		errs, err := service.ValidateEventAgainstSchema(b.Payload, schema)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"valid": len(errs) == 0, "errors": errs})
		return
	}

	switch r.Method {
	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		list, err := h.Svc.ListSchemas(tenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		filter := strings.ToLower(r.URL.Query().Get("filter")); if filter != "" { filtered := list[:0]; for _, item := range list { if strings.Contains(strings.ToLower(item.Name), filter) { filtered = append(filtered, item) } }; list = filtered }
		sort.SliceStable(list, func(i, j int) bool { if r.URL.Query().Get("dir") == "asc" { return list[i].Version < list[j].Version }; return list[i].Version > list[j].Version })
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
