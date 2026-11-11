package service

import (
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestQueryServiceCachesFilterResultsBriefly(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Event{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Event{EventID: "one", EventName: "click", ClientTS: 1})
	q := &QueryService{DB: db}
	first, err := q.Filter(EventFilter{EventName: "click"}, 10, 0)
	if err != nil || len(first) != 1 {
		t.Fatalf("first query: %v len=%d", err, len(first))
	}
	db.Create(&model.Event{EventID: "two", EventName: "click", ClientTS: 2})
	second, err := q.Filter(EventFilter{EventName: "click"}, 10, 0)
	if err != nil || len(second) != 1 {
		t.Fatalf("expected cached result, err=%v len=%d", err, len(second))
	}
}

func TestQueryServiceFilterScopesPostgresReadTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Event{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Event{EventID: "tenant-event", TenantID: "tenant-a", EventName: "click", ClientTS: 1})
	var scopedTenant string
	q := &QueryService{DB: db, SetTenantContext: func(tx *gorm.DB, tenantID string) error {
		if tx != db { // the read must run on the transaction, not the pool handle
			scopedTenant = tenantID
		}
		return nil
	}}
	list, err := q.Filter(EventFilter{TenantID: "tenant-a", EventName: "click"}, 10, 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("scoped query: %v len=%d", err, len(list))
	}
	if scopedTenant != "tenant-a" {
		t.Fatalf("expected tenant context tenant-a, got %q", scopedTenant)
	}
}
