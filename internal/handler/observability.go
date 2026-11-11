package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/metrics"
)

type traceResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *traceResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *traceResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

// Observability adds a low-cardinality request metric and a correlation ID.
func Observability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 16)
		_, _ = rand.Read(buf)
		traceID := hex.EncodeToString(buf)
		w.Header().Set("X-Trace-ID", traceID)
		rw := &traceResponseWriter{ResponseWriter: w}
		started := time.Now()
		next.ServeHTTP(rw, r)
		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, fmt.Sprint(status)).Observe(time.Since(started).Seconds())
	})
}
