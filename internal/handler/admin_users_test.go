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
	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest("GET", "/admin-users?page=true&limit=1&offset=0&sort=email&dir=asc", nil))
	if page.Code != 200 || !bytes.Contains(page.Body.Bytes(), []byte(`"items"`)) || !bytes.Contains(page.Body.Bytes(), []byte(`"total":1`)) {
		t.Fatalf("paginated list response=%d body=%s", page.Code, page.Body.String())
	}
	remove := httptest.NewRecorder()
	h.ServeHTTP(remove, httptest.NewRequest("DELETE", "/admin-users/"+created.ID, nil))
	if remove.Code != 204 {
		t.Fatalf("delete status=%d", remove.Code)
	}
}

func TestAdminInvitationCanBeAcceptedOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AdminUser{}); err != nil {
		t.Fatal(err)
	}
	h := &AdminUsersHandler{DB: db}
	create := httptest.NewRecorder()
	h.ServeHTTP(create, httptest.NewRequest("POST", "/admin-users", bytes.NewBufferString(`{"email":"invite@example.com"}`)))
	var response struct {
		model.AdminUser
		InvitationToken string `json:"invitation_token"`
	}
	if err := json.NewDecoder(create.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.InvitationToken == "" || response.Status != "invited" {
		t.Fatalf("unexpected invitation response: %+v", response)
	}
	accept := httptest.NewRecorder()
	(&AdminInvitationHandler{DB: db}).ServeHTTP(accept, httptest.NewRequest("POST", "/admin-users/accept", bytes.NewBufferString(`{"token":"`+response.InvitationToken+`"}`)))
	if accept.Code != 200 {
		t.Fatalf("accept status=%d body=%s", accept.Code, accept.Body.String())
	}
	acceptAgain := httptest.NewRecorder()
	(&AdminInvitationHandler{DB: db}).ServeHTTP(acceptAgain, httptest.NewRequest("POST", "/admin-users/accept", bytes.NewBufferString(`{"token":"`+response.InvitationToken+`"}`)))
	if acceptAgain.Code != 401 {
		t.Fatalf("second accept status=%d", acceptAgain.Code)
	}
}

func TestAdminTeamsManageMemberships(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminTeam{}, &model.AdminTeamMember{}); err != nil {
		t.Fatal(err)
	}
	user := model.AdminUser{ID: "u1", Email: "a@example.com", Role: "analyst", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	h := &AdminTeamsHandler{DB: db}
	create := httptest.NewRecorder()
	h.ServeHTTP(create, httptest.NewRequest("POST", "/admin-teams", bytes.NewBufferString(`{"name":"Growth"}`)))
	if create.Code != 201 {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var team model.AdminTeam
	if err := json.NewDecoder(create.Body).Decode(&team); err != nil {
		t.Fatal(err)
	}
	add := httptest.NewRecorder()
	h.ServeHTTP(add, httptest.NewRequest("POST", "/admin-teams/"+team.ID+"/members", bytes.NewBufferString(`{"user_id":"u1","role":"editor"}`)))
	if add.Code != 201 {
		t.Fatalf("add status=%d body=%s", add.Code, add.Body.String())
	}
	list := httptest.NewRecorder()
	h.ServeHTTP(list, httptest.NewRequest("GET", "/admin-teams/"+team.ID+"/members", nil))
	if list.Code != 200 || !bytes.Contains(list.Body.Bytes(), []byte(`"user_id":"u1"`)) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
}
