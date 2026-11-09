package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

// LiveSessionHandler serves a Server-Sent Events stream of a session's
// running event count and last seen, bridging the browser (which can't speak
// gRPC) to the in-process LiveSessionHub (see internal/service/live_session.go).
// It sends an immediate snapshot, then one more event per subsequent
// ingested event for that session, until the client disconnects.
type LiveSessionHandler struct {
	Hub *service.LiveSessionHub
}

func (h *LiveSessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	ch, unsubscribe := h.Hub.Subscribe(sessionID)
	defer unsubscribe()

	send := func(sess *model.Session) bool {
		b, _ := json.Marshal(map[string]any{
			"event_count": sess.EventCount,
			"last_seen":   sess.LastSeen,
		})
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	snapshot := h.Hub.Snapshot(sessionID)
	if snapshot == nil {
		// Session not yet tracked — send zeros
		if !send(&model.Session{EventCount: 0}) {
			return
		}
	} else if !send(snapshot) {
		return
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case sess := <-ch:
			if !send(sess) {
				return
			}
		}
	}
}