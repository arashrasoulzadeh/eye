package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type ReplayHandler struct{ Svc *service.ReplayService }

func (h *ReplayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var b struct {
		TenantID string `json:"tenant_id"`
		FromTS   int64  `json:"from_ts"`
		ToTS     int64  `json:"to_ts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.TenantID == "" {
		http.Error(w, "tenant_id, from_ts, to_ts required", http.StatusBadRequest)
		return
	}
	replayed, err := h.Svc.Replay(b.TenantID, b.FromTS, b.ToTS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]int{"replayed": replayed})
}
