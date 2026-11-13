package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestInit(t *testing.T) {
	// Reset prometheus registry for testing
	prometheus.Unregister(StreamQueueDepth)
	prometheus.Unregister(IngestSampleRate)
	prometheus.Unregister(HTTPRequestDuration)
	prometheus.Unregister(IngestTotal)
	prometheus.Unregister(IngestLatency)
	prometheus.Unregister(IngestErrorsTotal)
	prometheus.Unregister(HTTPRequestsTotal)
	prometheus.Unregister(HTTPErrorsTotal)
	prometheus.Unregister(StoredEvents)
	prometheus.Unregister(StoredSessions)
	prometheus.Unregister(StoredTenants)
	prometheus.Unregister(StoredBuckets)
	prometheus.Unregister(IntelligenceProfiles)
	prometheus.Unregister(IntelligenceModels)
	prometheus.Unregister(IntelligenceTrainingRuns)
	prometheus.Unregister(DatabaseRows)
	prometheus.Unregister(HTTPRouteDuration)
	prometheus.Unregister(MLTrainingExamples)
	prometheus.Unregister(MLTrainingPositiveExamples)
	prometheus.Unregister(MLTrainingNegativeExamples)
	prometheus.Unregister(MLAnomalyScoreAverage)
	prometheus.Unregister(MLPredictionConfidenceAverage)

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
