package handler

import (
	"encoding/json"
	"net/http"

	"github.com/eye/eye/server/internal/service"
)

type IngestHandler struct{ Svc *service.IngestService }

func (h *IngestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	a, rej := h.Svc.Ingest(body.Events)
	json.NewEncoder(w).Encode(map[string]int{"accepted": a, "rejected": rej})
}
