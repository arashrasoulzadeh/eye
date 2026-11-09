package service

import (
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func TestLiveSessionHub(t *testing.T) {
	hub := NewLiveSessionHub()

	// Test Update and Snapshot
	sess := &model.Session{ID: "s1", EventCount: 5, LastSeen: time.Now()}
	hub.Update(sess)

	snapshot := hub.Snapshot("s1")
	if snapshot == nil {
		t.Fatal("Snapshot returned nil")
	}
	if snapshot.EventCount != 5 {
		t.Errorf("EventCount = %d, want 5", snapshot.EventCount)
	}

	// Test non-existent session
	if hub.Snapshot("nonexistent") != nil {
		t.Error("expected nil for non-existent session")
	}

	// Test AddEvent and EventSnapshot
	event := &model.Event{EventID: "e1", SessionID: "s1", EventName: "click"}
	hub.AddEvent("s1", event)

	events := hub.EventSnapshot("s1", 50)
	if len(events) != 1 {
		t.Errorf("EventSnapshot len = %d, want 1", len(events))
	}
	if events[0].EventID != "e1" {
		t.Errorf("EventID = %s, want e1", events[0].EventID)
	}

	// Test multiple events and limit
	for i := 0; i < 105; i++ {
		hub.AddEvent("s1", &model.Event{EventID: string(rune(i)), SessionID: "s1"})
	}
	events = hub.EventSnapshot("s1", 100)
	if len(events) != 100 {
		t.Errorf("EventSnapshot len = %d, want 100", len(events))
	}

	// Test Subscribe and unsubscribe
	ch, unsub := hub.Subscribe("s1")
	select {
	case <-ch:
		t.Error("unexpected initial send")
	default:
	}
	unsub()

	// After unsubscribe, should not receive
	sess2 := &model.Session{ID: "s1", EventCount: 10}
	hub.Update(sess2)
	select {
	case <-ch:
		t.Error("received after unsubscribe")
	case <-time.After(10 * time.Millisecond):
		// expected
	}

	// Test SubscribeEvents
	ch2, unsub2 := hub.SubscribeEvents("s1")
	hub.AddEvent("s1", &model.Event{EventID: "new"})
	select {
	case evs := <-ch2:
		if len(evs) != 1 || evs[0].EventID != "new" {
			t.Errorf("unexpected event: %v", evs)
		}
	default:
		t.Error("expected event on channel")
	}
	unsub2()
}