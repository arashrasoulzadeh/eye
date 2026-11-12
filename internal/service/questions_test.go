package service

import (
	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestQuestionCRUDAndRelativeResults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:questions-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Question{}, &model.Event{}); err != nil {
		t.Fatal(err)
	}
	svc := &QuestionService{DB: db}
	q, err := svc.Save(&model.Question{TenantID: "t1", Name: "Signups", EventName: "signup", ChartType: "line", RangeMode: "relative", RelativeDays: 7})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for i := 0; i < 2; i++ {
		if err := db.Create(&model.Event{EventID: questionID(), TenantID: "t1", EventName: "signup", ClientTS: now.Add(-time.Duration(i) * 24 * time.Hour).UnixMilli()}).Error; err != nil {
			t.Fatal(err)
		}
	}
	result, err := svc.Results("t1", q.ID, now, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Points) != 2 {
		t.Fatalf("points = %#v", result.Points)
	}
	if err := svc.Delete("t1", q.ID); err != nil {
		t.Fatal(err)
	}
	list, err := svc.List("t1")
	if err != nil || len(list) != 0 {
		t.Fatalf("list after delete = %#v, %v", list, err)
	}
}

func TestQuestionListPageFiltersAndOrders(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Question{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Question{ID: "one", TenantID: "t1", Name: "Signup", EventNames: "signup", UpdatedAt: 1})
	db.Create(&model.Question{ID: "two", TenantID: "t1", Name: "Purchase", EventNames: "purchase", UpdatedAt: 2})
	items, total, err := (&QuestionService{DB: db}).ListPage("t1", "purchase", "desc", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != "two" {
		t.Fatalf("items=%+v total=%d", items, total)
	}
}
