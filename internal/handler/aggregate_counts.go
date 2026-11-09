package handler

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

// AggregateCountsHandler reads back the Phase 7 materialized per-(tenant,
// event_name, dims) counts maintained by the stream pipeline's consumer.
type AggregateCountsHandler struct{ DB *gorm.DB }

func (h *AggregateCountsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tenantID, eventName, dims := q.Get("tenant_id"), q.Get("event_name"), q.Get("dims")
	if tenantID == "" || eventName == "" {
		http.Error(w, "tenant_id and event_name are required", 400)
		return
	}
	count, err := service.GetAggregateCount(h.DB, tenantID, eventName, dims)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"tenant_id": tenantID, "event_name": eventName, "dims": dims, "count": count})
}
