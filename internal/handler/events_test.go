package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEventsHandlerCursorPagination(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Event{}); err != nil {
		t.Fatal(err)
	}
	base := time.Now()
	for i := 0; i < 3; i++ {
		db.Create(&model.Event{EventID: string(rune('a' + i)), IngestTS: base.Add(time.Duration(i) * time.Second)})
	}
	h := &EventsHandler{DB: db}
	req := httptest.NewRequest(http.MethodGet, "/events?limit=2", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK || res.Header().Get("X-Next-Cursor") == "" {
		t.Fatalf("status=%d cursor=%q", res.Code, res.Header().Get("X-Next-Cursor"))
	}
	req = httptest.NewRequest(http.MethodGet, "/events?limit=2&cursor="+res.Header().Get("X-Next-Cursor"), nil)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK || res.Header().Get("X-Next-Cursor") != "" {
		t.Fatalf("unexpected second page: status=%d cursor=%q", res.Code, res.Header().Get("X-Next-Cursor"))
	}
}
