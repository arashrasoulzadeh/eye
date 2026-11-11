package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminUsersHandlerCRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AdminUser{}); err != nil {
		t.Fatal(err)
	}
	h := &AdminUsersHandler{DB: db}
	create := httptest.NewRecorder()
	h.ServeHTTP(create, httptest.NewRequest("POST", "/admin-users", bytes.NewBufferString(`{"email":"a@example.com","name":"A","role":"admin"}`)))
	if create.Code != 201 {
		t.Fatalf("create status=%d", create.Code)
	}
	var created model.AdminUser
	if err := json.NewDecoder(create.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Status != "invited" || created.Role != "admin" {
		t.Fatalf("unexpected user: %+v", created)
	}
	update := httptest.NewRecorder()
	h.ServeHTTP(update, httptest.NewRequest("PUT", "/admin-users/"+created.ID, bytes.NewBufferString(`{"email":"a@example.com","name":"Updated","role":"owner","status":"active"}`)))
	if update.Code != 200 {
		t.Fatalf("update status=%d", update.Code)
	}
	list := httptest.NewRecorder()
	h.ServeHTTP(list, httptest.NewRequest("GET", "/admin-users?filter=Updated", nil))
	if list.Code != 200 || !bytes.Contains(list.Body.Bytes(), []byte("Updated")) {
		t.Fatalf("list response=%d body=%s", list.Code, list.Body.String())
	}
	remove := httptest.NewRecorder()
	h.ServeHTTP(remove, httptest.NewRequest("DELETE", "/admin-users/"+created.ID, nil))
	if remove.Code != 204 {
		t.Fatalf("delete status=%d", remove.Code)
	}
}
