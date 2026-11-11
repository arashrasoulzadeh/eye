package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTenantDataHandlerRequiresConfirmationAndDeletesScopedData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Tenant{}, &model.Event{}, &model.Session{}, &model.Schema{}, &model.Question{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Tenant{ID: "t1", Name: "one"})
	db.Create(&model.Tenant{ID: "t2", Name: "two"})
	db.Create(&model.Event{EventID: "e1", TenantID: "t1"})
	db.Create(&model.Event{EventID: "e2", TenantID: "t2"})
	h := &TenantDataHandler{DB: db}
	noConfirm := httptest.NewRecorder()
	h.ServeHTTP(noConfirm, httptest.NewRequest(http.MethodDelete, "/tenants/data/t1", nil))
	if noConfirm.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", noConfirm.Code)
	}
	req := httptest.NewRequest(http.MethodDelete, "/tenants/data/t1?confirm=true", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	var left []model.Event
	db.Find(&left)
	if len(left) != 1 || left[0].TenantID != "t2" {
		t.Fatalf("unexpected remaining events: %+v", left)
	}
}
