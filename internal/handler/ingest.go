package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/metrics"
)

// Ingester is satisfied by both the synchronous IngestService and the async
// StreamIngestPipeline, so IngestHandler can front either.
type Ingester interface {
	Ingest(events []map[string]any) (int, int)
}

// Sampler decides, per event, whether to admit it into the ingest path at
// all. Satisfied by *service.SamplingController; left nil (the zero value
// for a plain IngestHandler) disables sampling entirely so every existing
// caller/test that doesn't care about Phase 8 is unaffected.
type Sampler interface {
	Accept() bool
}

// Limiter is satisfied by *service.LiveRateLimiter. Nil (the zero value)
// means unlimited, so every existing caller/test that doesn't set it is
// unaffected.
type Limiter interface {
	Allow() bool
}

type IngestHandler struct {
	Svc Ingester
	// Sampler, if set, is consulted once per event before it reaches Svc.
	// A dropped event counts toward "rejected" in the response and toward
	// IngestTotal{status="sampled"} rather than IngestErrorsTotal, since it's
	// an intentional load-shedding decision, not a failure.
	Sampler Sampler
	// Limiter, if set, gates the whole request before decoding. Rejected
	// requests get a 429, matching the separate collector binary's existing
	// per-tenant rate-limit behavior, but driven by live admin settings.
	Limiter Limiter
}

func (h *IngestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if h.Limiter != nil && !h.Limiter.Allow() {
		metrics.IngestErrorsTotal.WithLabelValues("rate_limited").Inc()
		metrics.IngestLatency.Observe(time.Since(start).Seconds())
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	var body struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("decode").Inc()
		metrics.IngestLatency.Observe(time.Since(start).Seconds())
		http.Error(w, err.Error(), 400)
		return
	}
	events := body.Events
	sampled := 0
	if h.Sampler != nil {
		kept := make([]map[string]any, 0, len(events))
		for _, e := range events {
			if h.Sampler.Accept() {
				kept = append(kept, e)
			} else {
				sampled++
			}
		}
		events = kept
	}
	a, rej := h.Svc.Ingest(events)
	metrics.IngestTotal.WithLabelValues("accepted").Add(float64(a))
	metrics.IngestTotal.WithLabelValues("rejected").Add(float64(rej))
	if sampled > 0 {
		metrics.IngestTotal.WithLabelValues("sampled").Add(float64(sampled))
	}
	metrics.IngestLatency.Observe(time.Since(start).Seconds())
	json.NewEncoder(w).Encode(map[string]int{"accepted": a, "rejected": rej + sampled})
}
