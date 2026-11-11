package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/metrics"
)

func TestSLOHandlerReportsAvailabilityAndBudget(t *testing.T) {
	metrics.RecordHTTPRequest(http.StatusOK)
	metrics.RecordHTTPRequest(http.StatusInternalServerError)
	w := httptest.NewRecorder()
	SLOHandler{Target: 0.9}.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/slo", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["errors"].(float64) < 1 || body["error_budget_remaining"].(float64) < 0 {
		t.Fatalf("unexpected slo response: %#v", body)
	}
}
