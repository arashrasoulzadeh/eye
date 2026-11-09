package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/eye/eye/internal/metrics"
)

// Ingester is satisfied by both the synchronous IngestService and the async
// StreamIngestPipeline, so IngestHandler can front either.
type Ingester interface {
	Ingest(events []map[string]any) (int, int)
}

type IngestHandler struct{ Svc Ingester }

func (h *IngestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var body struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("decode").Inc()
		metrics.IngestLatency.Observe(time.Since(start).Seconds())
		http.Error(w, err.Error(), 400)
		return
	}
	a, rej := h.Svc.Ingest(body.Events)
	metrics.IngestTotal.WithLabelValues("accepted").Add(float64(a))
	metrics.IngestTotal.WithLabelValues("rejected").Add(float64(rej))
	metrics.IngestLatency.Observe(time.Since(start).Seconds())
	json.NewEncoder(w).Encode(map[string]int{"accepted": a, "rejected": rej})
}
