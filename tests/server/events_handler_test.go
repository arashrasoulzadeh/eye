package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/repository"
)

func newEventsTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	if err := r.Migrate(); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestEventsHandlerFiltersByTenantAndSession(t *testing.T) {
	db := newEventsTestDB(t)
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click"})
	db.Create(&model.Event{EventID: "e2", SessionID: "s2", TenantID: "t1", EventName: "click"})
	db.Create(&model.Event{EventID: "e3", SessionID: "s1", TenantID: "t2", EventName: "view"})

	h := &handler.EventsHandler{DB: db}

	req := httptest.NewRequest(http.MethodGet, "/events?tenant_id=t1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var list []model.Event
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 events for tenant_id=t1, got %d", len(list))
	}

	req2 := httptest.NewRequest(http.MethodGet, "/events?session_id=s1", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	var list2 []model.Event
	if err := json.NewDecoder(w2.Body).Decode(&list2); err != nil {
		t.Fatal(err)
	}
	if len(list2) != 2 {
		t.Fatalf("expected 2 events for session_id=s1, got %d", len(list2))
	}

	req3 := httptest.NewRequest(http.MethodGet, "/events?tenant_id=t1&session_id=s1", nil)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, req3)
	var list3 []model.Event
	if err := json.NewDecoder(w3.Body).Decode(&list3); err != nil {
		t.Fatal(err)
	}
	if len(list3) != 1 || list3[0].EventID != "e1" {
		t.Fatalf("expected exactly event e1 for tenant_id=t1&session_id=s1, got %d results", len(list3))
	}
}
