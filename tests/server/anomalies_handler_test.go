package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/service"
)

func TestAnomaliesHandlerReturnsFlaggedPairs(t *testing.T) {
	db := newAnomalyTestDB(t)
	svc := service.NewAnomalyService(db)
	svc.WindowSize = 3

	db.Create(&model.AggregateCount{TenantID: "t1", EventName: "click", DimsKey: "", Count: 0})
	cumulative := int64(0)
	for i, d := range []int64{10, 10, 10, 10, 10, 900} {
		cumulative += d
		db.Model(&model.AggregateCount{}).Where("tenant_id = ?", "t1").Update("count", cumulative)
		if err := svc.Sample(time.Unix(int64(i)*300, 0)); err != nil {
			t.Fatalf("sample %d: %v", i, err)
		}
	}

	h := &handler.AnomaliesHandler{Svc: svc}
	req := httptest.NewRequest(http.MethodGet, "/anomalies", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got []service.Anomaly
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].TenantID != "t1" || got[0].EventName != "click" {
		t.Fatalf("unexpected anomalies response: %+v", got)
	}
}

func TestAnomaliesHandlerReturnsEmptyArrayWhenNoneFlagged(t *testing.T) {
	db := newAnomalyTestDB(t)
	svc := service.NewAnomalyService(db)

	h := &handler.AnomaliesHandler{Svc: svc}
	req := httptest.NewRequest(http.MethodGet, "/anomalies", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if body != "[]\n" && body != "[]" {
		t.Fatalf("expected an empty JSON array, got %q", body)
	}
}
