package test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/handler"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

var errTimeout = errors.New("timed out waiting for SSE line")

// startSSERequest fires the handler in a goroutine against a ResponseRecorder
// wired through an io.Pipe, so the test can read Server-Sent Events as they
// are flushed rather than waiting for the handler to return (it never does
// until the request context is canceled).
func startSSERequest(t *testing.T, h http.Handler, url string) (*bufio.Reader, func()) {
	t.Helper()
	pr, pw := io.Pipe()
	req := httptest.NewRequest("GET", url, nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	rec := &pipeResponseWriter{pw: pw, header: http.Header{}}

	go func() {
		h.ServeHTTP(rec, req)
		pw.Close()
	}()

	return bufio.NewReader(pr), cancel
}

// pipeResponseWriter adapts an io.PipeWriter to http.ResponseWriter so the
// handler's writes are observable as they happen (http.ResponseRecorder only
// buffers, it can't be read mid-flight).
type pipeResponseWriter struct {
	pw          io.WriteCloser
	header      http.Header
	wroteHeader bool
}

func (w *pipeResponseWriter) Header() http.Header { return w.header }
func (w *pipeResponseWriter) Write(b []byte) (int, error) {
	return w.pw.Write(b)
}
func (w *pipeResponseWriter) WriteHeader(code int) { w.wroteHeader = true }
func (w *pipeResponseWriter) Flush()               {}

func TestLiveCountersHandlerStreamsSnapshotThenUpdates(t *testing.T) {
	hub := service.NewLiveCounterHub()
	hub.Increment("tenant-a")
	h := &handler.LiveCountersHandler{Hub: hub}

	reader, cancel := startSSERequest(t, h, "/live-counters?tenant_id=tenant-a")
	defer cancel()

	line, err := readSSEDataLine(reader, 2*time.Second)
	if err != nil {
		t.Fatalf("reading initial snapshot: %v", err)
	}
	if !strings.Contains(line, `"count":1`) {
		t.Fatalf("expected initial snapshot count 1, got %q", line)
	}

	hub.Increment("tenant-a")
	line, err = readSSEDataLine(reader, 2*time.Second)
	if err != nil {
		t.Fatalf("reading live update: %v", err)
	}
	if !strings.Contains(line, `"count":2`) {
		t.Fatalf("expected live update count 2, got %q", line)
	}
}

func readSSEDataLine(r *bufio.Reader, timeout time.Duration) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				ch <- result{"", err}
				return
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				ch <- result{line, nil}
				return
			}
		}
	}()
	select {
	case res := <-ch:
		return res.line, res.err
	case <-time.After(timeout):
		return "", errTimeout
	}
}
