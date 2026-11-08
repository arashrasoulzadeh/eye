package handler

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type SessionsHandler struct{ DB *gorm.DB }

func (h *SessionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var list []model.Session
	h.DB.Find(&list)
	json.NewEncoder(w).Encode(list)
}
