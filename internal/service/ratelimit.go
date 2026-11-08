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
