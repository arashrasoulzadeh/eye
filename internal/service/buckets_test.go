package service

import (
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBucketServiceGroupsAndCachesCounts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Event{}, &model.EventBucket{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Event{EventID: "a", TenantID: "t1", EventName: "click", ClientTS: 61_000})
	db.Create(&model.Event{EventID: "b", TenantID: "t1", EventName: "click", ClientTS: 119_000})
	db.Create(&model.Event{EventID: "c", TenantID: "t1", EventName: "click", ClientTS: 601_000})
	svc := &BucketService{DB: db}
	var events []model.Event
	db.Find(&events)
	for i := range events {
		if err := UpsertEventBuckets(db, &events[i]); err != nil {
			t.Fatal(err)
		}
	}
	points, err := svc.Events("t1", "click", "10minute", 0, 900_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 || points[0].Bucket != 0 || points[0].Count != 2 || points[1].Bucket != 600_000 || points[1].Count != 1 {
		t.Fatalf("unexpected buckets: %#v", points)
	}
	db.Create(&model.Event{EventID: "other", TenantID: "t2", EventName: "click", ClientTS: 118_000})
	for _, scope := range []struct {
		tenant string
		count  int
	}{{"t1", 1}, {"", 2}} {
		bounded, err := svc.Events(scope.tenant, "click", "10minute", 100_000, 119_000)
		if err != nil || len(bounded) != 1 || bounded[0].Count != scope.count {
			t.Fatalf("tenant=%q buckets=%v err=%v", scope.tenant, bounded, err)
		}
	}
	db.Create(&model.Event{EventID: "d", TenantID: "t1", EventName: "click", ClientTS: 120_000})
	cached, err := svc.Events("t1", "click", "10minute", 0, 900_000)
	if err != nil || len(cached) != 2 || cached[0].Count != 2 {
		t.Fatalf("expected cached buckets, err=%v points=%#v", err, cached)
	}
}
