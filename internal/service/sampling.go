package service

import (
	"math/rand"
	"sync"
)

// SamplingController computes an adaptive ingest sample rate from a load
// signal (the Phase 7 ChannelStream's queue depth) and decides, per event,
// whether to accept or drop it at that rate. It exists so /ingest degrades
// gracefully under load instead of falling over: as the queue backs up, an
// increasing fraction of new events are dropped before they add to the
// backlog, and the rate recovers automatically once load subsides.
//
// Thresholds are deliberately simple and queue-depth-based rather than any
// kind of learned/ML policy — "adaptive sampling" here means "a sample rate
// that adapts to an observed load signal," not a model.
type SamplingController struct {
	mu   sync.Mutex
	rate float64

	// Thresholds, inclusive lower bounds, evaluated low depth first. Below
	// Low, rate is 1.0 (no sampling). Between Low and Mid, rate is Mid.
	// Between Mid and High, rate is MidRate. At/above High, rate is LowRate.
	Low, Mid, High          int
	MidRate, LowRate        float64

	// Rand is the decision source for Accept, injected so tests are
	// deterministic instead of depending on real randomness. Defaults to a
	// package-level *rand.Rand seeded at construction if left nil.
	Rand *rand.Rand
}

// NewSamplingController returns a controller with reasonable default
// thresholds: under 50 queued messages, accept everything; 50-199, accept
// 50%; 200+, accept 10%.
func NewSamplingController() *SamplingController {
	return &SamplingController{
		Low: 50, Mid: 200, High: 200,
		MidRate: 0.5, LowRate: 0.1,
		rate: 1.0,
		Rand: rand.New(rand.NewSource(1)),
	}
}

// UpdateRate recomputes the current sample rate from a fresh load reading
// (e.g. ChannelStream.QueueDepth()) and returns it.
func (s *SamplingController) UpdateRate(queueDepth int) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case queueDepth >= s.High:
		s.rate = s.LowRate
	case queueDepth >= s.Low:
		s.rate = s.MidRate
	default:
		s.rate = 1.0
	}
	return s.rate
}

// Rate returns the current sample rate without recomputing it.
func (s *SamplingController) Rate() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rate
}

// Accept decides whether one event should be accepted at the current sample
// rate. At rate 1.0 it always accepts (so normal load is never affected by
// the injected RNG at all). Thread-safe.
func (s *SamplingController) Accept() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rate >= 1.0 {
		return true
	}
	if s.rate <= 0 {
		return false
	}
	return s.Rand.Float64() < s.rate
}
