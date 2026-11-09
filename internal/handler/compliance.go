package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/eye/eye/internal/service"
)

// ComplianceHandler exposes GDPR-style erasure at DELETE /compliance/erase.
// A DELETE body is unusual but matches the admin API's existing habit of
// JSON bodies on write methods (see TenantsHandler) rather than encoding
// the session id in the path.
type ComplianceHandler struct{ Svc *service.ComplianceService }

func (h *ComplianceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var b struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.SessionID == "" {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}
	deleted, err := h.Svc.EraseSession(b.SessionID)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"session_id":     b.SessionID,
		"events_deleted": deleted,
	})
}
