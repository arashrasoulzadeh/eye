package handler

import (
	"encoding/json"
	"net/http"
)

// Ingester is satisfied by both the synchronous IngestService and the async
// StreamIngestPipeline, so IngestHandler can front either.
type Ingester interface {
	Ingest(events []map[string]any) (int, int)
}

type IngestHandler struct{ Svc Ingester }

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
