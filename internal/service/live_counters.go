package service

import "sync"

// LiveCounterHub tracks a running per-tenant event count and broadcasts each
// update to subscribers, backing the LiveCounters gRPC stream. It's plain
// in-memory state, scoped to one server process — fine for a single-node
// dev deployment, same honest scope as ChannelStream (see stream.go).
type LiveCounterHub struct {
	mu    sync.Mutex
	count map[string]int64
	subs  map[string][]chan int64
}

func NewLiveCounterHub() *LiveCounterHub {
	return &LiveCounterHub{
		count: make(map[string]int64),
		subs:  make(map[string][]chan int64),
	}
}

// Increment bumps tenantID's running count by one and notifies its
// subscribers. A subscriber channel that's momentarily full (consumer lag)
// drops this update rather than blocking ingestion — subscribers should
// read Snapshot for the authoritative current value.
func (h *LiveCounterHub) Increment(tenantID string) {
	h.mu.Lock()
	h.count[tenantID]++
	count := h.count[tenantID]
	subs := append([]chan int64(nil), h.subs[tenantID]...)
	h.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- count:
		default:
		}
	}
}

// Snapshot returns tenantID's current count (0 if never incremented).
func (h *LiveCounterHub) Snapshot(tenantID string) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count[tenantID]
}

// Subscribe registers a channel that receives the updated count after every
// Increment for tenantID. Call the returned unsubscribe func (e.g. via
// defer) once done, typically when the gRPC stream's context is canceled.
func (h *LiveCounterHub) Subscribe(tenantID string) (chan int64, func()) {
	ch := make(chan int64, 8)
	h.mu.Lock()
	h.subs[tenantID] = append(h.subs[tenantID], ch)
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subs := h.subs[tenantID]
		for i, c := range subs {
			if c == ch {
				h.subs[tenantID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
	return ch, unsubscribe
}
