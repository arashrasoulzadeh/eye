package service

import (
	"sync"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// LiveSessionHub tracks per-session event count, last seen, and events,
// broadcasting updates to subscribers. It's plain in-memory state, scoped to
// one server process — fine for a single-node dev deployment.
type LiveSessionHub struct {
	mu         sync.Mutex
	sessions   map[string]*model.Session
	events     map[string][]*model.Event
	subs       map[string][]chan *model.Session
	eventSubs  map[string][]chan []*model.Event
}

func NewLiveSessionHub() *LiveSessionHub {
	return &LiveSessionHub{
		sessions:  make(map[string]*model.Session),
		events:    make(map[string][]*model.Event),
		subs:      make(map[string][]chan *model.Session),
		eventSubs: make(map[string][]chan []*model.Event),
	}
}

// Update updates the session's EventCount and LastSeen, then notifies
// subscribers. A subscriber channel that's momentarily full (consumer lag)
// drops this update rather than blocking ingestion — subscribers should
// read Snapshot for the authoritative current value.
func (h *LiveSessionHub) Update(session *model.Session) {
	h.mu.Lock()
	h.sessions[session.ID] = session
	subs := append([]chan *model.Session(nil), h.subs[session.ID]...)
	h.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- session:
		default:
		}
	}
}

// AddEvent adds an event to the session's event log and notifies event subscribers.
func (h *LiveSessionHub) AddEvent(sessionID string, event *model.Event) {
	h.mu.Lock()
	// Keep last 100 events per session
	evs := h.events[sessionID]
	evs = append(evs, event)
	if len(evs) > 100 {
		evs = evs[len(evs)-100:]
	}
	h.events[sessionID] = evs
	eventSubs := append([]chan []*model.Event(nil), h.eventSubs[sessionID]...)
	h.mu.Unlock()

	for _, ch := range eventSubs {
		select {
		case ch <- []*model.Event{event}:
		default:
		}
	}
}

// Snapshot returns sessionID's current session (nil if never updated).
func (h *LiveSessionHub) Snapshot(sessionID string) *model.Session {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[sessionID]
}

// EventSnapshot returns the last n events for a session.
func (h *LiveSessionHub) EventSnapshot(sessionID string, n int) []*model.Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	evs := h.events[sessionID]
	if len(evs) > n {
		return evs[len(evs)-n:]
	}
	return evs
}

// Subscribe registers a channel that receives the updated session after every
// Update for sessionID. Call the returned unsubscribe func (e.g. via
// defer) once done, typically when the SSE stream's context is canceled.
func (h *LiveSessionHub) Subscribe(sessionID string) (chan *model.Session, func()) {
	ch := make(chan *model.Session, 8)
	h.mu.Lock()
	h.subs[sessionID] = append(h.subs[sessionID], ch)
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subs := h.subs[sessionID]
		for i, c := range subs {
			if c == ch {
				h.subs[sessionID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
	return ch, unsubscribe
}

// SubscribeEvents registers a channel that receives new events after every
// AddEvent for sessionID. Call the returned unsubscribe func when done.
func (h *LiveSessionHub) SubscribeEvents(sessionID string) (chan []*model.Event, func()) {
	ch := make(chan []*model.Event, 8)
	h.mu.Lock()
	h.eventSubs[sessionID] = append(h.eventSubs[sessionID], ch)
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subs := h.eventSubs[sessionID]
		for i, c := range subs {
			if c == ch {
				h.eventSubs[sessionID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
	return ch, unsubscribe
}