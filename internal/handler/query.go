package handler

import (
	"encoding/json"
	"net/http"

	"github.com/eye/eye/internal/service"
)

type QueryHandler struct{ Svc *service.QueryService }

func (h *QueryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("event_name")
	list, _ := h.Svc.EventsByName(name, 100)
	json.NewEncoder(w).Encode(list)
}
