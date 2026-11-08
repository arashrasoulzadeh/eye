package metrics

import "github.com/prometheus/client_golang/prometheus"

var IngestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ingest_events_total"}, []string{"status"})
var IngestLatency = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "ingest_latency_seconds"})

func Init() {
	prometheus.MustRegister(IngestTotal, IngestLatency)
}
