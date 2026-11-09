package test

import (
	"math/rand"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

func TestSamplingControllerRateThresholds(t *testing.T) {
	sc := service.NewSamplingController()

	if got := sc.UpdateRate(0); got != 1.0 {
		t.Fatalf("low load: expected rate 1.0, got %v", got)
	}
	if got := sc.UpdateRate(49); got != 1.0 {
		t.Fatalf("just under Low threshold: expected rate 1.0, got %v", got)
	}
	if got := sc.UpdateRate(50); got != 0.5 {
		t.Fatalf("at Low threshold: expected rate 0.5, got %v", got)
	}
	if got := sc.UpdateRate(199); got != 0.5 {
		t.Fatalf("just under High threshold: expected rate 0.5, got %v", got)
	}
	if got := sc.UpdateRate(200); got != 0.1 {
		t.Fatalf("at High threshold: expected rate 0.1, got %v", got)
	}
	if got := sc.UpdateRate(10000); got != 0.1 {
		t.Fatalf("far above High threshold: expected rate 0.1, got %v", got)
	}
}

func TestSamplingControllerRateRecovers(t *testing.T) {
	sc := service.NewSamplingController()
	sc.UpdateRate(500)
	if got := sc.Rate(); got != 0.1 {
		t.Fatalf("expected dropped rate 0.1 under load, got %v", got)
	}
	sc.UpdateRate(0)
	if got := sc.Rate(); got != 1.0 {
		t.Fatalf("expected rate to recover to 1.0 once load clears, got %v", got)
	}
}

func TestSamplingControllerAcceptAlwaysAcceptsAtFullRate(t *testing.T) {
	sc := service.NewSamplingController()
	sc.UpdateRate(0)
	for i := 0; i < 1000; i++ {
		if !sc.Accept() {
			t.Fatal("expected Accept() to always return true at rate 1.0")
		}
	}
}

func TestSamplingControllerAcceptMatchesConfiguredRateStatistically(t *testing.T) {
	sc := service.NewSamplingController()
	sc.Rand = rand.New(rand.NewSource(42))
	sc.UpdateRate(500) // -> LowRate 0.1

	const trials = 20000
	accepted := 0
	for i := 0; i < trials; i++ {
		if sc.Accept() {
			accepted++
		}
	}
	got := float64(accepted) / float64(trials)
	if got < 0.08 || got > 0.12 {
		t.Fatalf("expected acceptance rate near 0.1 over %d trials, got %v", trials, got)
	}
}

func TestSamplingControllerAcceptNeverAcceptsAtZeroRate(t *testing.T) {
	sc := service.NewSamplingController()
	sc.LowRate = 0
	sc.UpdateRate(500)
	for i := 0; i < 1000; i++ {
		if sc.Accept() {
			t.Fatal("expected Accept() to always return false at rate 0")
		}
	}
}
