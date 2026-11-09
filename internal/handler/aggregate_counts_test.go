package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func newTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.AggregateCount{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAggregateCountsHandler_ServeHTTP(t *testing.T) {
	db := newTestDB(t)
	h := &AggregateCountsHandler{DB: db}

	// Setup: create an aggregate count
	if err := service.UpsertAggregateCount(db, "t1", "click", `{"platform":"web"}`); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	tests := []struct {
		name           string
		query          string
		wantStatus     int
		wantCount      int64
		wantError      bool
	}{
		{
			name:       "valid request",
			query:      "?tenant_id=t1&event_name=click&dims=%7B%22platform%22%3A%22web%22%7D",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "missing tenant_id",
			query:      "?event_name=click",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "missing event_name",
			query:      "?tenant_id=t1",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "non-existent aggregate",
			query:      "?tenant_id=t1&event_name=purchase",
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/aggregate-counts"+tc.query, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			if tc.wantError {
				return
			}

			var resp map[string]any
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}

			if resp["tenant_id"] != "t1" {
				t.Errorf("tenant_id = %v, want t1", resp["tenant_id"])
			}
if resp["event_name"] != tc.query[strings.Index(tc.query, "event_name=")+len("event_name="):] {
			// Extract event_name from query for comparison
			if !strings.Contains(tc.query, "event_name=purchase") {
				if resp["event_name"] != "click" {
					t.Errorf("event_name = %v, want click", resp["event_name"])
				}
			}
		}
			if resp["count"] != float64(tc.wantCount) {
				t.Errorf("count = %v, want %d", resp["count"], tc.wantCount)
			}
		})
	}
}