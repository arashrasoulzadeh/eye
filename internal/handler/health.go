package handler

import (
	"encoding/json"
	"net/http"
)

// HealthHandler is intentionally unauthenticated so reverse proxies and
// container orchestrators can check readiness without possessing admin or
// tenant credentials.
type HealthHandler struct{}

func (HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
