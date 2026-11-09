package service

import (
	"sync"
	"time"
)

type Bucket struct {
	mu       sync.Mutex
	tokens   float64
	last     time.Time
	rate     float64
	capacity float64
}

func NewBucket(rate, capacity float64) *Bucket {
	return &Bucket{tokens: capacity, last: time.Now(), rate: rate, capacity: capacity}
}

func (b *Bucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.tokens += b.rate * now.Sub(b.last).Seconds()
	if b.tokens > b.capacity { b.tokens = b.capacity }
	b.last = now
	if b.tokens < 1 { return false }
	b.tokens--
	return true
}

// RateProvider is satisfied by *SettingsService: it's the live source of
// truth LiveRateLimiter consults on every call, so a PUT /settings change
// takes effect on the very next ingest request rather than at restart.
type RateProvider interface {
	CurrentRateLimit() float64
}

// LiveRateLimiter fronts the REST ingest path with a single global token
// bucket whose rate is re-read from RateProvider on every call. Unlike the
// separate collector binary's per-tenant Bucket (constructed once with a
// hardcoded rate at startup — see server/cmd/collector/main.go), this one
// is genuinely adjustable at runtime: a rate change rebuilds the bucket
// (losing accumulated burst capacity, which is an acceptable, documented
// trade-off for a settings change taking effect immediately).
type LiveRateLimiter struct {
	Provider RateProvider

	mu       sync.Mutex
	bucket   *Bucket
	lastRate float64
}

func (l *LiveRateLimiter) Allow() bool {
	rate := l.Provider.CurrentRateLimit()
	if rate <= 0 {
		rate = 50
	}
	l.mu.Lock()
	if l.bucket == nil || l.lastRate != rate {
		l.bucket = NewBucket(rate, rate*2)
		l.lastRate = rate
	}
	b := l.bucket
	l.mu.Unlock()
	return b.Allow()
}
