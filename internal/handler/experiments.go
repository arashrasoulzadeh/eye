package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/eye/eye/internal/service"
)

type ExperimentsHandler struct{ Svc *service.ExperimentService }

func (h *ExperimentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/experiments")
	path = strings.Trim(path, "/")

	if r.Method == http.MethodGet && strings.HasSuffix(path, "/assign") {
		experimentID := strings.TrimSuffix(path, "/assign")
		experimentID = strings.TrimSuffix(experimentID, "/")
		sessionID := r.URL.Query().Get("session_id")
		if experimentID == "" || sessionID == "" {
			http.Error(w, "experiment id and session_id are required", http.StatusBadRequest)
			return
		}
		variant, err := h.Svc.AssignVariant(experimentID, sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"experiment_id": experimentID, "variant": variant})
		return
	}

	switch r.Method {
	case http.MethodPost:
		var b struct {
			Name     string
			TenantID string
			Variants string
			Traffic  int
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		exp, err := h.Svc.Create(b.Name, b.TenantID, b.Variants, b.Traffic)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(exp)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
