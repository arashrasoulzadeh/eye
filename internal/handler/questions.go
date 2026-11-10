package handler

import (
	"encoding/json"
	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
	"net/http"
	"strings"
	"time"
)

type QuestionsHandler struct{ Svc *service.QuestionService }

func (h *QuestionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/questions"), "/")
	tenant := r.URL.Query().Get("tenant_id")
	if r.Method == http.MethodGet && strings.HasSuffix(path, "/results") {
		id := strings.TrimSuffix(path, "/results")
		result, err := h.Svc.Results(tenant, id, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(result)
		return
	}
	if r.Method == http.MethodDelete && path != "" {
		if err := h.Svc.Delete(tenant, path); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	switch r.Method {
	case http.MethodGet:
		list, err := h.Svc.List(tenant)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(list)
	case http.MethodPost, http.MethodPut:
		var q model.Question
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if r.Method == http.MethodPut && q.ID == "" {
			q.ID = path
		}
		saved, err := h.Svc.Save(&q)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		json.NewEncoder(w).Encode(saved)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
