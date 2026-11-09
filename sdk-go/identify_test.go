package eyesdk

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIdentifyMergesUserIDAndTraitsIntoDims(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"accepted":1,"rejected":0}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, Options{TenantID: "t1", MaxBufferSize: 10, BatchSize: 0, FlushInterval: time.Hour})
	c.SetSessionID("s1")
	c.Identify("user-42", map[string]any{"plan": "pro"})

	if err := c.Track("click", nil, map[string]any{"button": "cta"}); err != nil {
		t.Fatal(err)
	}
	buf := c.PeekBuffer()
	if len(buf) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(buf))
	}
	dims := buf[0].Dims
	if dims["user_id"] != "user-42" {
		t.Errorf("expected dims.user_id=user-42, got %v", dims["user_id"])
	}
	if dims["plan"] != "pro" {
		t.Errorf("expected dims.plan=pro (from traits), got %v", dims["plan"])
	}
	if dims["button"] != "cta" {
		t.Errorf("expected dims.button=cta (from the Track call) to survive merging, got %v", dims["button"])
	}
}

func TestTrackWithoutIdentifyLeavesDimsUntouched(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()

	c := newTestClient(t, srv, Options{TenantID: "t1", MaxBufferSize: 10, BatchSize: 0, FlushInterval: time.Hour})
	c.SetSessionID("s1")

	if err := c.Track("click", nil, map[string]any{"button": "cta"}); err != nil {
		t.Fatal(err)
	}
	dims := c.PeekBuffer()[0].Dims
	if _, hasUserID := dims["user_id"]; hasUserID {
		t.Error("expected no user_id in dims when Identify was never called")
	}
	if len(dims) != 1 || dims["button"] != "cta" {
		t.Errorf("expected dims unchanged at {button: cta}, got %v", dims)
	}
}
