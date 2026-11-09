package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

// LiveEventsHandler serves a Server-Sent Events stream of events for a session,
// bridging the browser (which can't speak gRPC) to the in-process LiveSessionHub.
// It sends an immediate snapshot of recent events, then one more event per
// subsequent ingested event for that session, until the client disconnects.
type LiveEventsHandler struct {
	Hub *service.LiveSessionHub
}

func (h *LiveEventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, unsubscribe := h.Hub.SubscribeEvents(sessionID)
	defer unsubscribe()

	send := func(events []*model.Event) bool {
		b, _ := json.Marshal(map[string]any{"events": events})
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	// Send initial snapshot of recent events (last 50)
	snapshot := h.Hub.EventSnapshot(sessionID, 50)
	if !send(snapshot) {
		return
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case events := <-ch:
			if !send(events) {
				return
			}
		}
	}
}