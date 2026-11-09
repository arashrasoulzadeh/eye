package service

import "sync"

// TierRates maps a tenant's Tier field to its ingest events/sec limit.
// An unknown/empty tier falls back to "free" via rateForTier below, so
// tenants created before Phase 14 (which all have Tier == "") keep working
// unchanged at the lowest tier rather than erroring.
var TierRates = map[string]float64{
	"free":       5,
	"pro":        50,
	"enterprise": 500,
}

func rateForTier(tier string) float64 {
	if r, ok := TierRates[tier]; ok {
		return r
	}
	return TierRates["free"]
}

// TenantRateLimiter gives each tenant its own token bucket, sized by that
// tenant's Tier, rather than one global rate shared by everyone (the
// LiveRateLimiter in ratelimit.go). It's consulted per-event inside
// IngestService.Ingest, after the tenant has been resolved from the
// session, so a free-tier tenant bursting past its rate only rejects its
// own events — it can't starve other tenants and isn't throttled by them.
type TenantRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tenantBucket
}

type tenantBucket struct {
	bucket *Bucket
	tier   string
}

func NewTenantRateLimiter() *TenantRateLimiter {
	return &TenantRateLimiter{buckets: make(map[string]*tenantBucket)}
}

// Allow reports whether tenantID may ingest one more event right now,
// given its current tier. Changing a tenant's tier takes effect on its
// next bucket rebuild (immediately, since the tier is checked on every
// call) rather than requiring a process restart.
func (l *TenantRateLimiter) Allow(tenantID, tier string) bool {
	rate := rateForTier(tier)
	l.mu.Lock()
	tb, ok := l.buckets[tenantID]
	if !ok || tb.tier != tier {
		tb = &tenantBucket{bucket: NewBucket(rate, rate*2), tier: tier}
		l.buckets[tenantID] = tb
	}
	b := tb.bucket
	l.mu.Unlock()
	return b.Allow()
}
