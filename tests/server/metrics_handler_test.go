package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/handler"
	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
)

func newMetricsTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	if err := r.Migrate(); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMetricsHandlerReturnsLiveCounts(t *testing.T) {
	db := newMetricsTestDB(t)
	db.Create(&model.Tenant{ID: "t1", Name: "acme"})
	db.Create(&model.Tenant{ID: "t2", Name: "globex"})
	db.Create(&model.Session{ID: "s1", TenantID: "t1"})
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click"})
	db.Create(&model.Event{EventID: "e2", SessionID: "s1", TenantID: "t1", EventName: "view"})

	h := &handler.MetricsHandler{DB: db}
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var got map[string]int64
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["events"] != 2 {
		t.Errorf("events = %d, want 2", got["events"])
	}
	if got["sessions"] != 1 {
		t.Errorf("sessions = %d, want 1", got["sessions"])
	}
	if got["tenants"] != 2 {
		t.Errorf("tenants = %d, want 2", got["tenants"])
	}
}

func TestMetricsHandlerZeroStateIsAllZeroes(t *testing.T) {
	db := newMetricsTestDB(t)
	h := &handler.MetricsHandler{DB: db}
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var got map[string]int64
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	for k, v := range got {
		if v != 0 {
			t.Errorf("%s = %d, want 0 on an empty DB", k, v)
		}
	}
}
