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
