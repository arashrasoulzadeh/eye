package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

type WebhookDeadLetterHandler struct{ Svc *service.WebhookDeadLetterService }

func (h *WebhookDeadLetterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	list, err := h.Svc.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(list)
}
