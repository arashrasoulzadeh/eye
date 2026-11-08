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
	"github.com/eye/eye/internal/service"
)

func newQueryTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	if err := r.Migrate(); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestQueryHandlerFiltersByEventName(t *testing.T) {
	db := newQueryTestDB(t)
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click"})
	db.Create(&model.Event{EventID: "e2", SessionID: "s1", TenantID: "t1", EventName: "view"})
	db.Create(&model.Event{EventID: "e3", SessionID: "s1", TenantID: "t1", EventName: "click"})

	h := &handler.QueryHandler{Svc: &service.QueryService{DB: db}}
	req := httptest.NewRequest(http.MethodGet, "/query?event_name=click", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var list []model.Event
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 click events, got %d", len(list))
	}
	for _, e := range list {
		if e.EventName != "click" {
			t.Errorf("unexpected event in results: %+v", e)
		}
	}
}

func TestQueryHandlerNoMatchesReturnsEmptyList(t *testing.T) {
	db := newQueryTestDB(t)
	h := &handler.QueryHandler{Svc: &service.QueryService{DB: db}}
	req := httptest.NewRequest(http.MethodGet, "/query?event_name=nope", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var list []model.Event
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no results, got %d", len(list))
	}
}

func TestQueryHandlerFiltersByTenantSessionAndTimeRangeWithPagination(t *testing.T) {
	db := newQueryTestDB(t)
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: 100})
	db.Create(&model.Event{EventID: "e2", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: 200})
	db.Create(&model.Event{EventID: "e3", SessionID: "s2", TenantID: "t1", EventName: "click", ClientTS: 150})
	db.Create(&model.Event{EventID: "e4", SessionID: "s1", TenantID: "t2", EventName: "click", ClientTS: 150})

	h := &handler.QueryHandler{Svc: &service.QueryService{DB: db}}

	req := httptest.NewRequest(http.MethodGet, "/query?tenant_id=t1&session_id=s1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var list []model.Event
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("tenant_id=t1&session_id=s1: expected 2, got %d", len(list))
	}

	req2 := httptest.NewRequest(http.MethodGet, "/query?tenant_id=t1&from_ts=120&to_ts=160", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	var list2 []model.Event
	if err := json.NewDecoder(w2.Body).Decode(&list2); err != nil {
		t.Fatal(err)
	}
	if len(list2) != 1 || list2[0].EventID != "e3" {
		t.Fatalf("time range filter: expected exactly e3, got %+v", list2)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/query?tenant_id=t1&limit=1&offset=1", nil)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, req3)
	var list3 []model.Event
	if err := json.NewDecoder(w3.Body).Decode(&list3); err != nil {
		t.Fatal(err)
	}
	if len(list3) != 1 {
		t.Fatalf("pagination: expected 1, got %d", len(list3))
	}
}
