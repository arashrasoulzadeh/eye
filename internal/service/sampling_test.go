package service

import (
	"testing"
)

func TestSamplingController(t *testing.T) {
	sc := NewSamplingController()

	// Test UpdateRate with various queue depths
	// Thresholds: Low=50, Mid=200, High=200, MidRate=0.5, LowRate=0.1
	tests := []struct {
		depth  int
		expect float64
	}{
		{0, 1.0},     // < Low, full rate
		{10, 1.0},    // < Low, full rate
		{50, 0.5},    // >= Low, MidRate
		{100, 0.5},   // >= Low, MidRate
		{199, 0.5},   // >= Low, MidRate
		{200, 0.1},   // >= High, LowRate
		{500, 0.1},   // >= High, LowRate
		{1000, 0.1},  // >= High, LowRate
		{10000, 0.1}, // >= High, LowRate
	}

	for _, tc := range tests {
		rate := sc.UpdateRate(tc.depth)
		if rate < 0 || rate > 1 {
			t.Errorf("UpdateRate(%d) = %f, want 0-1", tc.depth, rate)
		}
		if rate != tc.expect {
			t.Errorf("UpdateRate(%d) = %f, want %f", tc.depth, rate, tc.expect)
		}
	}

	// Test Rate() returns last computed rate
	sc.UpdateRate(100)
	rate := sc.Rate()
	if rate != 0.5 {
		t.Errorf("Rate() = %f, want 0.5", rate)
	}

	// Test Accept with different rates
	sc.UpdateRate(0) // rate = 1.0
	accepted := 0
	for i := 0; i < 1000; i++ {
		if sc.Accept() {
			accepted++
		}
	}
	if accepted != 1000 {
		t.Errorf("Accept() accepted %d/1000 at rate 1.0, want 1000", accepted)
	}

	// Test Accept with rate 0.5
	sc.UpdateRate(100) // rate = 0.5
	accepted = 0
	for i := 0; i < 1000; i++ {
		if sc.Accept() {
			accepted++
		}
	}
	// Should accept roughly 500 (allow some variance)
	if accepted < 450 || accepted > 550 {
		t.Errorf("Accept() accepted %d/1000 at rate 0.5, want ~500", accepted)
	}

	// Test Accept with rate 0.1
	sc.UpdateRate(500) // rate = 0.1
	accepted = 0
	for i := 0; i < 1000; i++ {
		if sc.Accept() {
			accepted++
		}
	}
	// Should accept roughly 100 (allow some variance)
	if accepted < 50 || accepted > 150 {
		t.Errorf("Accept() accepted %d/1000 at rate 0.1, want ~100", accepted)
	}
}