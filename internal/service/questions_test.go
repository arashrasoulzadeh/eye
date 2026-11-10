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
