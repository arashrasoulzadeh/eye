package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEntityResponsesIncludeIntelligenceTags(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Event{}, &model.IntelligenceInference{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{
		&model.Session{ID: "session", TenantID: "tenant", UserID: "user"},
		&model.Event{EventID: "event", TenantID: "tenant", SessionID: "session"},
		&model.IntelligenceInference{TenantID: "tenant", Kind: "session", EntityID: "session", Tags: `["gameplay"]`},
		&model.IntelligenceInference{TenantID: "tenant", Kind: "event", EntityID: "event", Tags: `["entry_point"]`},
		&model.IntelligenceInference{TenantID: "tenant", Kind: "user", EntityID: "user", Tags: `["returning"]`},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := &service.UserIntelligenceService{DB: db}
	sessions := &SessionsHandler{DB: db, Intelligence: svc}
	events := &EventsHandler{DB: db, Intelligence: svc}
	query := &QueryHandler{Svc: &service.QueryService{DB: db}, Intelligence: svc}
	for _, tc := range []struct {
		handler http.Handler
		path    string
		want    []string
	}{
		{sessions, "/sessions?session_id=session", []string{`"intelligence_tags":["gameplay"]`}},
		{sessions, "/sessions?include_user_tags=true", []string{`"intelligence_tags":["gameplay"]`, `"user_intelligence_tags":["returning"]`}},
		{events, "/events/event", []string{`"intelligence_tags":["entry_point"]`}},
		{query, "/query?page=true", []string{`"intelligence_tags":["entry_point"]`}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d: %s", rec.Code, rec.Body)
			}
			for _, want := range tc.want {
				if !strings.Contains(rec.Body.String(), want) {
					t.Fatalf("missing %s: %s", want, rec.Body)
				}
			}
		})
	}
}
