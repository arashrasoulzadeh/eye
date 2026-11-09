package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/handler"
	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func TestCohortsHandlerCreateAndList(t *testing.T) {
	db := newCohortTestDB(t)
	h := &handler.CohortsHandler{Svc: &service.CohortService{DB: db}}

	body, _ := json.Marshal(map[string]any{"tenant_id": "t1", "name": "power_clickers", "event_name": "click", "min_count": 3})
	req := httptest.NewRequest(http.MethodPost, "/cohorts", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", w.Code, w.Body.String())
	}
	var created model.Cohort
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("expected a generated ID")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/cohorts?tenant_id=t1", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	var list []model.Cohort
	if err := json.NewDecoder(w2.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "power_clickers" {
		t.Fatalf("expected exactly the created cohort, got %+v", list)
	}
}

func TestCohortsHandlerMaterializeAndMembers(t *testing.T) {
	db := newCohortTestDB(t)
	svc := &service.CohortService{DB: db}
	c, _ := svc.Create("t1", "power_clickers", "click", 1)
	db.Create(&model.Event{EventID: "e1", TenantID: "t1", SessionID: "s1", EventName: "click"})

	h := &handler.CohortsHandler{Svc: svc}

	req := httptest.NewRequest(http.MethodPost, "/cohorts/"+c.ID+"/materialize", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("materialize status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["members"] != 1 {
		t.Fatalf("expected 1 member, got %+v", resp)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/cohorts/"+c.ID+"/members", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	var members []string
	if err := json.NewDecoder(w2.Body).Decode(&members); err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0] != "s1" {
		t.Fatalf("expected [s1], got %v", members)
	}
}
