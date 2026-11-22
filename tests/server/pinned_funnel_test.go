package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/handler"
	"github.com/arashrasoulzadeh/eye/internal/model"
)

func TestPinnedFunnelPersistsOneOrderedDefinitionPerTenant(t *testing.T) {
	db := newAnalyticsTestDB(t)
	if err := db.AutoMigrate(&model.Tenant{}, &model.PinnedFunnel{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		if err := db.Create(&model.Tenant{ID: id, Name: id, APIKey: id}).Error; err != nil {
			t.Fatal(err)
		}
	}
	call := func(method, tenant, body string, status int, expected string) {
		t.Helper()
		h := &handler.PinnedFunnelHandler{DB: db}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(method, "/analytics/funnel/pinned?tenant_id="+tenant, strings.NewReader(body)))
		if w.Code != status || !strings.Contains(w.Body.String(), expected) {
			t.Fatalf("%s %s: status = %d, body = %q", method, tenant, w.Code, w.Body.String())
		}
	}
	call(http.MethodGet, "a", "", 200, "null")
	call(http.MethodPost, "a", `{"name":"Signup","steps":["view","signup"]}`, 200, `"steps":["view","signup"]`)
	call(http.MethodGet, "a", "", 200, `"name":"Signup"`)
	call(http.MethodGet, "b", "", 200, "null")
	call(http.MethodPost, "a", `{"name":"Buy","steps":["signup","purchase"]}`, 200, `"name":"Buy"`)
	call(http.MethodPost, "a", `{"name":"Invalid","steps":["signup","signup"]}`, 400, "")
	call(http.MethodGet, "a", "", 200, `"steps":["signup","purchase"]`)
	call(http.MethodPost, "missing", `{"name":"Buy","steps":["signup","purchase"]}`, 404, "")
	call(http.MethodDelete, "a", "", 204, "")
	call(http.MethodGet, "a", "", 200, "null")
}
