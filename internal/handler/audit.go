package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func randomAuditID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// auditRecordingResponseWriter captures the status code the wrapped handler
// actually wrote, so the audit entry reflects what happened rather than
// assuming 200.
type auditRecordingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *auditRecordingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Audit wraps next so every non-GET request that reaches it is recorded to
// the AuditLog table after the handler completes. It must be placed INSIDE
// AdminAuth (i.e. AdminAuth(key, Audit(db, next))) so a request that fails
// authentication never reaches this middleware and is never logged as an
// action — only requests that actually passed admin auth are audited.
//
// This project has a single shared admin API key with no per-user
// identity, so ActorKey is always the fixed label "admin": there is no way
// to tell which human used the key, and this middleware does not pretend
// otherwise.
func Audit(db *gorm.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		rw := &auditRecordingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		entry := &model.AuditLog{
			ID:         randomAuditID(),
			Action:     r.Method,
			Resource:   r.URL.Path,
			ActorKey:   "admin",
			StatusCode: rw.status,
			CreatedAt:  time.Now().Unix(),
		}
		_ = db.Create(entry).Error
	})
}
