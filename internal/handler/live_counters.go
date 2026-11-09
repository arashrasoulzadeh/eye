package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/eye/eye/internal/service"
)

// LiveCountersHandler serves a Server-Sent Events stream of a tenant's
// running event count, bridging the browser (which can't speak gRPC)
// to the in-process LiveCounterHub (see internal/service/live_counters.go).
// It sends an immediate snapshot, then one more event per subsequent
// ingested event for that tenant, until the client disconnects.
type LiveCountersHandler struct {
	Hub *service.LiveCounterHub
}

func (h *LiveCountersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
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

	ch, unsubscribe := h.Hub.Subscribe(tenantID)
	defer unsubscribe()

	send := func(count int64) bool {
		b, _ := json.Marshal(map[string]int64{"count": count})
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !send(h.Hub.Snapshot(tenantID)) {
		return
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case count := <-ch:
			if !send(count) {
				return
			}
		}
	}
}
