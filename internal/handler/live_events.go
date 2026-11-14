package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

// LiveEventsHandler serves a Server-Sent Events stream of events for a session,
// bridging the browser (which can't speak gRPC) to the in-process LiveSessionHub.
// It sends an immediate snapshot of recent events, then one more event per
// subsequent ingested event for that session, until the client disconnects.
type LiveEventsHandler struct {
	Hub          *service.LiveSessionHub
	Intelligence *service.UserIntelligenceService
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

	tenant := r.URL.Query().Get("tenant_id")
	from, _ := strconv.ParseInt(r.URL.Query().Get("from_ts"), 10, 64)
	to, _ := strconv.ParseInt(r.URL.Query().Get("to_ts"), 10, 64)
	send := func(events []*model.Event) bool {
		rows := make([]model.Event, 0, len(events))
		for _, event := range events {
			if event != nil && (tenant == "" || event.TenantID == tenant) && (from == 0 || event.ClientTS >= from) && (to == 0 || event.ClientTS <= to) {
				rows = append(rows, *event)
			}
		}
		b, _ := json.Marshal(map[string]any{"events": attachEventTags(r.Context(), h.Intelligence, rows)})
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
