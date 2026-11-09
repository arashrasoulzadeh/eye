package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

type SettingsHandler struct{ Svc *service.SettingsService }

type settingsBody struct {
	RetentionHours     int `json:"retention_hours"`
	RateLimitPerSecond int `json:"rate_limit_per_second"`
}

func (h *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s, err := h.Svc.Get()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(s)
	case http.MethodPut:
		var b settingsBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s, err := h.Svc.Update(model.RuntimeSettings{
			RetentionHours:     b.RetentionHours,
			RateLimitPerSecond: b.RateLimitPerSecond,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(s)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
