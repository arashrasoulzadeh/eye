package test

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
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

func TestQueryServiceFilterSortDescOrdersNewestFirst(t *testing.T) {
	db := newQueryServiceFilterTestDB(t)
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: 100})
	db.Create(&model.Event{EventID: "e2", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: 300})
	db.Create(&model.Event{EventID: "e3", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: 200})

	svc := &service.QueryService{DB: db}
	list, err := svc.Filter(service.EventFilter{TenantID: "t1", SortDesc: true}, 100, 0)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 events, got %d", len(list))
	}
	got := []string{list[0].EventID, list[1].EventID, list[2].EventID}
	want := []string{"e2", "e3", "e1"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected newest-first order %v, got %v", want, got)
		}
	}
}

func TestQueryServiceFilterDefaultsToAscending(t *testing.T) {
	db := newQueryServiceFilterTestDB(t)
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: 300})
	db.Create(&model.Event{EventID: "e2", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: 100})

	svc := &service.QueryService{DB: db}
	list, err := svc.Filter(service.EventFilter{TenantID: "t1"}, 100, 0)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if len(list) != 2 || list[0].EventID != "e2" || list[1].EventID != "e1" {
		t.Fatalf("expected unchanged ascending default order, got %+v", list)
	}
}

func newQueryServiceFilterTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := db.AutoMigrate(&model.Event{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestQueryServiceFilterByTenantSessionAndEventName(t *testing.T) {
	db := newQueryServiceFilterTestDB(t)
	db.Create(&model.Event{EventID: "e1", TenantID: "t1", SessionID: "s1", EventName: "click", ClientTS: 100})
	db.Create(&model.Event{EventID: "e2", TenantID: "t1", SessionID: "s2", EventName: "click", ClientTS: 100})
	db.Create(&model.Event{EventID: "e3", TenantID: "t2", SessionID: "s1", EventName: "click", ClientTS: 100})
	db.Create(&model.Event{EventID: "e4", TenantID: "t1", SessionID: "s1", EventName: "view", ClientTS: 100})

	svc := &service.QueryService{DB: db}

	list, err := svc.Filter(service.EventFilter{TenantID: "t1"}, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("tenant_id=t1: expected 3, got %d", len(list))
	}

	list, err = svc.Filter(service.EventFilter{TenantID: "t1", SessionID: "s1"}, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("tenant_id=t1&session_id=s1: expected 2, got %d", len(list))
	}

	list, err = svc.Filter(service.EventFilter{TenantID: "t1", SessionID: "s1", EventName: "click"}, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].EventID != "e1" {
		t.Fatalf("expected exactly e1, got %+v", list)
	}
}

func TestQueryServiceFilterByTimeRange(t *testing.T) {
	db := newQueryServiceFilterTestDB(t)
	db.Create(&model.Event{EventID: "e1", TenantID: "t1", SessionID: "s1", EventName: "click", ClientTS: 1000})
	db.Create(&model.Event{EventID: "e2", TenantID: "t1", SessionID: "s1", EventName: "click", ClientTS: 2000})
	db.Create(&model.Event{EventID: "e3", TenantID: "t1", SessionID: "s1", EventName: "click", ClientTS: 3000})

	svc := &service.QueryService{DB: db}
	from := int64(1500)
	to := int64(2500)
	list, err := svc.Filter(service.EventFilter{TenantID: "t1", FromTS: &from, ToTS: &to}, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].EventID != "e2" {
		t.Fatalf("expected exactly e2 within [1500,2500], got %+v", list)
	}
}

func TestQueryServiceFilterByDims(t *testing.T) {
	db := newQueryServiceFilterTestDB(t)
	db.Create(&model.Event{EventID: "e1", TenantID: "t1", SessionID: "s1", EventName: "click", Dims: `{"platform":"ios"}`})
	db.Create(&model.Event{EventID: "e2", TenantID: "t1", SessionID: "s1", EventName: "click", Dims: `{"platform":"web"}`})

	svc := &service.QueryService{DB: db}
	list, err := svc.Filter(service.EventFilter{TenantID: "t1", DimsContains: `"platform":"ios"`}, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].EventID != "e1" {
		t.Fatalf("expected exactly e1 for platform=ios, got %+v", list)
	}
}

func TestQueryServiceFilterPagination(t *testing.T) {
	db := newQueryServiceFilterTestDB(t)
	for i := 0; i < 5; i++ {
		db.Create(&model.Event{EventID: string(rune('a' + i)), TenantID: "t1", SessionID: "s1", EventName: "click", ClientTS: int64(i)})
	}

	svc := &service.QueryService{DB: db}
	page1, err := svc.Filter(service.EventFilter{TenantID: "t1"}, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1: expected 2, got %d", len(page1))
	}

	page2, err := svc.Filter(service.EventFilter{TenantID: "t1"}, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 {
		t.Fatalf("page2: expected 2, got %d", len(page2))
	}
	if page1[0].EventID == page2[0].EventID {
		t.Fatal("page1 and page2 should not overlap")
	}

	page3, err := svc.Filter(service.EventFilter{TenantID: "t1"}, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(page3) != 1 {
		t.Fatalf("page3 (tail): expected 1, got %d", len(page3))
	}
}

// Guard against the timestamp zero-value ambiguity: a filter with no
// From/To set must not accidentally exclude everything.
func TestQueryServiceFilterWithNoTimeRangeReturnsAll(t *testing.T) {
	db := newQueryServiceFilterTestDB(t)
	db.Create(&model.Event{EventID: "e1", TenantID: "t1", SessionID: "s1", EventName: "click", ClientTS: time.Now().UnixMilli()})
	svc := &service.QueryService{DB: db}
	list, err := svc.Filter(service.EventFilter{TenantID: "t1"}, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
}
