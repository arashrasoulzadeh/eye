package metrics

import "github.com/prometheus/client_golang/prometheus"

// IngestTotal counts ingest requests by outcome: "accepted"/"rejected" for
// events that made it through decoding (per-event, from the accepted/rejected
// tallies IngestHandler gets back from the Ingester), and "error" for
// requests that failed before any event-level accounting (e.g. bad JSON).
var IngestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ingest_events_total"}, []string{"status"})

// IngestLatency observes wall-clock time for one /ingest or /ingest/stream
// HTTP request (decode + Ingester.Ingest call), i.e. the ingest rate's
// counterpart "how long did it take" metric.
var IngestLatency = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "ingest_latency_seconds"})

// IngestErrorsTotal counts ingest requests that failed outright (couldn't be
// decoded/processed at all), labeled by a short reason so "errors" in the
// Phase 9 checklist has its own series distinct from a per-event "rejected"
// outcome (a rejected event, e.g. a duplicate, is not an error).
var IngestErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ingest_errors_total"}, []string{"reason"})

// StreamQueueDepth reports how many messages are currently buffered in the
// Phase 7 ChannelStream's channel — the in-process substitute for a real
// broker's queue depth (see internal/service/stream.go). Sampled
// periodically by a ticker in main.go rather than pushed synchronously,
// since nothing else needs a live push and a gauge is naturally a sampled
// value.
var StreamQueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{Name: "stream_queue_depth"})

// IngestSampleRate reports the Phase 8 SamplingController's current accept
// rate (1.0 normally, dropping as StreamQueueDepth rises) so the adaptive
// sampling behavior is directly observable, not just inferred from dropped
// counts.
var IngestSampleRate = prometheus.NewGauge(prometheus.GaugeOpts{Name: "ingest_sample_rate"})

var HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP request duration by method and status."}, []string{"method", "status"})

func Init() {
	prometheus.MustRegister(IngestTotal, IngestLatency, IngestErrorsTotal, StreamQueueDepth, IngestSampleRate, HTTPRequestDuration)
}
