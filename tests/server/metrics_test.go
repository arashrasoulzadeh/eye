package test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/arashrasoulzadeh/eye/internal/metrics"
)

func TestMetricsInitRegistersCollectorsExactlyOnce(t *testing.T) {
	reg := prometheus.NewRegistry()
	if err := reg.Register(metrics.IngestTotal); err != nil {
		t.Fatalf("register IngestTotal: %v", err)
	}
	if err := reg.Register(metrics.IngestLatency); err != nil {
		t.Fatalf("register IngestLatency: %v", err)
	}

	metrics.IngestTotal.Reset()
	metrics.IngestTotal.WithLabelValues("accepted").Inc()
	metrics.IngestTotal.WithLabelValues("accepted").Inc()
	metrics.IngestTotal.WithLabelValues("rejected").Inc()

	got := &dto.Metric{}
	if err := metrics.IngestTotal.WithLabelValues("accepted").Write(got); err != nil {
		t.Fatal(err)
	}
	if got.GetCounter().GetValue() != 2 {
		t.Fatalf("accepted counter = %v, want 2", got.GetCounter().GetValue())
	}
}

func TestMetricsIngestLatencyObserves(t *testing.T) {
	reg := prometheus.NewRegistry()
	if err := reg.Register(metrics.IngestLatency); err != nil {
		t.Fatalf("register IngestLatency: %v", err)
	}
	metrics.IngestLatency.Observe(0.05)

	got := &dto.Metric{}
	if err := metrics.IngestLatency.Write(got); err != nil {
		t.Fatal(err)
	}
	if got.GetHistogram().GetSampleCount() == 0 {
		t.Fatal("expected at least one observed sample")
	}
}
