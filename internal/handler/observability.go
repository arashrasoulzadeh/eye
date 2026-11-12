package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/metrics"
)

var traceParentPattern = regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`)

func traceParent(r *http.Request) (string, string) {
	if value := r.Header.Get("traceparent"); traceParentPattern.MatchString(value) {
		return value, value[3:35]
	}
	buf := make([]byte, 24)
	_, _ = rand.Read(buf)
	traceID := hex.EncodeToString(buf[:16])
	spanID := hex.EncodeToString(buf[16:])
	return "00-" + traceID + "-" + spanID + "-01", traceID
}

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
		parent, traceID := traceParent(r)
		w.Header().Set("traceparent", parent)
		w.Header().Set("X-Trace-ID", traceID)
		rw := &traceResponseWriter{ResponseWriter: w}
		started := time.Now()
		next.ServeHTTP(rw, r)
		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, fmt.Sprint(status)).Observe(time.Since(started).Seconds())
		metrics.RecordHTTPRequest(status)
	})
}
