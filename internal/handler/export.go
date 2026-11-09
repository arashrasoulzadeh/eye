package handler

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

// ExportHandler serves GET /export/events?tenant_id=...&format=csv|json&from_ts=&to_ts=&event_name=.
// It reuses QueryService's existing filter logic so export results are
// always scoped/consistent with the regular /query endpoint, and streams
// up to a generous page size rather than the default /query limit.
type ExportHandler struct{ Svc *service.QueryService }

func (h *ExportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	tenantID := q.Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	format := q.Get("format")
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		http.Error(w, "format must be json or csv", http.StatusBadRequest)
		return
	}

	f := service.EventFilter{
		TenantID:  tenantID,
		SessionID: q.Get("session_id"),
		EventName: q.Get("event_name"),
	}
	if v := q.Get("from_ts"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.FromTS = &n
		}
	}
	if v := q.Get("to_ts"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.ToTS = &n
		}
	}

	limit := 100000
	events, err := h.Svc.Filter(f, limit, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="events.json"`)
		json.NewEncoder(w).Encode(events)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="events.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"event_id", "session_id", "tenant_id", "event_name", "client_ts", "ingest_ts", "payload", "dims", "source"})
	for _, e := range events {
		_ = cw.Write([]string{
			e.EventID,
			e.SessionID,
			e.TenantID,
			e.EventName,
			strconv.FormatInt(e.ClientTS, 10),
			e.IngestTS.UTC().Format("2006-01-02T15:04:05Z07:00"),
			e.Payload,
			e.Dims,
			e.Source,
		})
	}
	cw.Flush()
}
