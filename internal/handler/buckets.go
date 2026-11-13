package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type BucketsHandler struct{ Svc *service.BucketService }

func (h *BucketsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	bucket := q.Get("bucket")
	if bucket == "" {
		bucket = "hour"
	}
	from, fromErr := strconv.ParseInt(q.Get("from_ts"), 10, 64)
	to, toErr := strconv.ParseInt(q.Get("to_ts"), 10, 64)
	if fromErr != nil || toErr != nil || from >= to {
		http.Error(w, "from_ts and to_ts must be valid increasing milliseconds", http.StatusBadRequest)
		return
	}
	points, err := h.Svc.Events(tenantID, q.Get("event_name"), bucket, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=5, stale-while-revalidate=10")
	json.NewEncoder(w).Encode(points)
}
