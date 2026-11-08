package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/service"
)

func TestQueryServiceEventsByName(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := db.AutoMigrate(&model.Event{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click"})
	db.Create(&model.Event{EventID: "e2", SessionID: "s1", TenantID: "t1", EventName: "view"})
	db.Create(&model.Event{EventID: "e3", SessionID: "s1", TenantID: "t1", EventName: "click"})

	svc := &service.QueryService{DB: db}
	list, err := svc.EventsByName("click", 100)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 click events, got %d", len(list))
	}
}

func TestQueryServiceEventsByNameRespectsLimit(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := db.AutoMigrate(&model.Event{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		db.Create(&model.Event{EventID: string(rune('a' + i)), SessionID: "s1", TenantID: "t1", EventName: "click"})
	}

	svc := &service.QueryService{DB: db}
	list, err := svc.EventsByName("click", 2)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected limit of 2, got %d", len(list))
	}
}
