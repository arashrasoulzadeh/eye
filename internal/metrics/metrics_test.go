package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestInit(t *testing.T) {
	// Reset prometheus registry for testing
	prometheus.Unregister(StreamQueueDepth)
	prometheus.Unregister(IngestSampleRate)

	Init()

	if StreamQueueDepth == nil {
		t.Fatal("StreamQueueDepth not initialized")
	}
	if IngestSampleRate == nil {
		t.Fatal("IngestSampleRate not initialized")
	}

	// Test that metrics can be set
	StreamQueueDepth.Set(42)
	IngestSampleRate.Set(0.5)
}