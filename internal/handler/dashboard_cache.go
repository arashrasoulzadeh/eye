package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

// DashboardCacheHandler serves cached dashboard metrics from the DashboardCache table.
type DashboardCacheHandler struct {
	Svc *service.DashboardCacheService
}

func (h *DashboardCacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cache, err := h.Svc.Get()
	if err != nil {
		http.Error(w, "cache not ready", http.StatusServiceUnavailable)
		return
	}

	// Return cached data in the same format as the original metrics handler
	json.NewEncoder(w).Encode(map[string]any{
		"events":            cache.Events,
		"sessions":          cache.Sessions,
		"tenants":           cache.Tenants,
		"events_by_hour":    json.RawMessage(cache.EventsByHour),
		"events_by_tenant":  json.RawMessage(cache.EventsByTenant),
		"cached_at":         cache.UpdatedAt,
	})
}