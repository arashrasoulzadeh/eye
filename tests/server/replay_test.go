package test

import (
	"encoding/json"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func newReplayTestDB(t *testing.T) (*gorm.DB, *repository.Repo) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	r := repository.New(db)
	if err := r.Migrate(); err != nil {
		t.Fatal(err)
	}
	return db, r
}

func TestReplayServiceReplaysOnlyEventsInRange(t *testing.T) {
	db, r := newReplayTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})

	ingest := &service.IngestService{Repo: r}
	original := []map[string]any{
		{"event_id": "e-in-range", "session_id": "s1", "event_name": "click", "client_ts": float64(1000), "payload": map[string]any{"a": 1}, "dims": map[string]any{"platform": "web"}},
		{"event_id": "e-out-of-range", "session_id": "s1", "event_name": "click", "client_ts": float64(9000), "payload": map[string]any{}, "dims": map[string]any{}},
	}
	accepted, _ := ingest.Ingest(original)
	if accepted != 2 {
		t.Fatalf("expected 2 accepted originals, got %d", accepted)
	}

	rs := &service.ReplayService{DB: db, Ingest: ingest}
	replayed, err := rs.Replay("t1", 500, 2000)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replayed != 1 {
		t.Fatalf("expected 1 replayed event, got %d", replayed)
	}

	var events []model.Event
	db.Where("session_id = ?", "s1").Order("client_ts asc").Find(&events)
	if len(events) != 3 {
		t.Fatalf("expected 3 events total (2 original + 1 replay), got %d: %+v", len(events), events)
	}

	var replayRows []model.Event
	db.Where("source = ?", "replay").Find(&replayRows)
	if len(replayRows) != 1 {
		t.Fatalf("expected exactly 1 row tagged source=replay, got %d", len(replayRows))
	}
	rr := replayRows[0]
	if rr.EventID == "e-in-range" {
		t.Fatal("replayed event must have a distinct event_id from the original")
	}
	if rr.ClientTS != 1000 {
		t.Fatalf("replay should preserve the original client_ts, got %d", rr.ClientTS)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(rr.Payload), &payload); err != nil {
		t.Fatalf("replayed payload not valid json: %v", err)
	}
	if payload["a"] != float64(1) {
		t.Fatalf("replayed payload lost data: %+v", payload)
	}

	// The out-of-range event must remain untouched (no replay row for it).
	var outOfRangeStillOriginal model.Event
	if err := db.First(&outOfRangeStillOriginal, "event_id = ?", "e-out-of-range").Error; err != nil {
		t.Fatalf("original out-of-range event missing: %v", err)
	}
	if outOfRangeStillOriginal.Source == "replay" {
		t.Fatal("out-of-range event should not have been replayed")
	}
}

func TestReplayServiceDoesNotReplayAlreadyReplayedEvents(t *testing.T) {
	db, r := newReplayTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	ingest := &service.IngestService{Repo: r}
	ingest.Ingest([]map[string]any{
		{"event_id": "e1", "session_id": "s1", "event_name": "click", "client_ts": float64(1000), "payload": map[string]any{}, "dims": map[string]any{}},
	})

	rs := &service.ReplayService{DB: db, Ingest: ingest}
	first, err := rs.Replay("t1", 0, 2000)
	if err != nil || first != 1 {
		t.Fatalf("first replay: got %d, err %v", first, err)
	}

	// Replaying the same range again should only replay the original
	// (source="client") event again, not the replay row it just created —
	// otherwise replay counts would compound every time it's run.
	second, err := rs.Replay("t1", 0, 2000)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if second != 1 {
		t.Fatalf("expected second replay to re-replay only the 1 original event, got %d", second)
	}

	var replayRows int64
	db.Model(&model.Event{}).Where("source = ?", "replay").Count(&replayRows)
	if replayRows != 2 {
		t.Fatalf("expected 2 replay rows after two replay runs, got %d", replayRows)
	}
}

func TestReplayServiceScopedToTenant(t *testing.T) {
	db, r := newReplayTestDB(t)
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	db.Create(&model.Session{ID: "s2", TenantID: "t2"})
	ingest := &service.IngestService{Repo: r}
	ingest.Ingest([]map[string]any{
		{"event_id": "e1", "session_id": "s1", "event_name": "click", "client_ts": float64(1000), "payload": map[string]any{}, "dims": map[string]any{}},
		{"event_id": "e2", "session_id": "s2", "event_name": "click", "client_ts": float64(1000), "payload": map[string]any{}, "dims": map[string]any{}},
	})

	rs := &service.ReplayService{DB: db, Ingest: ingest}
	replayed, err := rs.Replay("t1", 0, 2000)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replayed != 1 {
		t.Fatalf("expected 1 replayed event scoped to t1, got %d", replayed)
	}
	var t2Replays int64
	db.Model(&model.Event{}).Where("tenant_id = ? AND source = ?", "t2", "replay").Count(&t2Replays)
	if t2Replays != 0 {
		t.Fatalf("tenant t2 should not have been touched by t1's replay")
	}
}
