package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func TestIngestDedup(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	r.Migrate()
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	svc := &service.IngestService{Repo: r}
	ev := []map[string]any{{"event_id": "e1", "event_name": "x", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"}}
	a, _ := svc.Ingest(ev)
	if a != 1 { t.Fatalf("expected 1 accepted got %d", a) }
	a2, _ := svc.Ingest(ev)
	if a2 != 0 { t.Fatalf("expected 0 on dup got %d", a2) }
}

// Malformed events (missing/wrong-typed required fields) must be rejected,
// not panic the server — this is exactly the shape a buggy client sends,
// e.g. the real bug found testing the CDN SDK build against a live server:
// a missing "event_name" used to crash the whole process via a bare type
// assertion.
func TestIngestRejectsMalformedEventsWithoutPanicking(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	r.Migrate()
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	svc := &service.IngestService{Repo: r}

	cases := []map[string]any{
		{"session_id": "s1", "event_id": "e-missing-name", "client_ts": float64(1)},
		{"session_id": "s1", "event_id": "e-missing-ts", "event_name": "x"},
		{"session_id": "s1", "event_name": "x", "client_ts": float64(1)},
		{"event_id": "e-missing-session", "event_name": "x", "client_ts": float64(1)},
		{"session_id": "s1", "event_id": "e-wrong-ts-type", "event_name": "x", "client_ts": "not-a-number"},
	}
	for _, ev := range cases {
		a, rej := svc.Ingest([]map[string]any{ev})
		if a != 0 || rej != 1 {
			t.Fatalf("expected malformed event %+v to be rejected (0 accepted, 1 rejected), got accepted=%d rejected=%d", ev, a, rej)
		}
	}
}

// Phase 14: an IngestService with a LiveCounters hub attached must bump the
// tenant's running count on every accepted event, for the real-time gRPC
// counters stream — and must NOT bump it for rejected (malformed/dup) ones.
func TestIngestIncrementsLiveCounterOnAcceptedEventsOnly(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	r.Migrate()
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	hub := service.NewLiveCounterHub()
	svc := &service.IngestService{Repo: r, Counters: hub}

	a, rej := svc.Ingest([]map[string]any{
		{"event_id": "e1", "event_name": "x", "client_ts": float64(1), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"},
		{"event_id": "e2", "event_name": "x", "client_ts": float64(2), "payload": map[string]any{}, "dims": map[string]any{}, "session_id": "s1"},
		{"event_id": "e-bad", "client_ts": float64(3)}, // missing event_name: rejected
	})
	if a != 2 || rej != 1 {
		t.Fatalf("expected 2 accepted, 1 rejected, got accepted=%d rejected=%d", a, rej)
	}
	if got := hub.Snapshot("t1"); got != 2 {
		t.Fatalf("expected tenant t1 live count 2, got %d", got)
	}
}
