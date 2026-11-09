package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/service"
)

func newCohortTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	// A plain ":memory:" DSN gives each pooled connection its own empty DB;
	// pin to a single connection so a background goroutine (the
	// materialization job) sees the same schema/data as the test.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.Cohort{}, &model.CohortMembership{}, &model.Event{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCohortServiceCreateAndList(t *testing.T) {
	db := newCohortTestDB(t)
	svc := &service.CohortService{DB: db}

	c, err := svc.Create("t1", "power_clickers", "click", 3)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ID == "" {
		t.Fatal("expected a generated ID")
	}

	if _, err := svc.Create("t2", "other", "click", 1); err != nil {
		t.Fatalf("create for t2: %v", err)
	}

	list, err := svc.List("t1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Name != "power_clickers" {
		t.Fatalf("expected exactly t1's cohort, got %+v", list)
	}
}

func TestCohortServiceMaterializeComputesMembershipByEventCount(t *testing.T) {
	db := newCohortTestDB(t)
	svc := &service.CohortService{DB: db}

	c, err := svc.Create("t1", "power_clickers", "click", 3)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// s1: 3 clicks -> qualifies. s2: 2 clicks -> doesn't. s3: 3 clicks for a
	// different tenant -> must not count.
	for i := 0; i < 3; i++ {
		db.Create(&model.Event{EventID: idFor("s1", i), TenantID: "t1", SessionID: "s1", EventName: "click"})
	}
	for i := 0; i < 2; i++ {
		db.Create(&model.Event{EventID: idFor("s2", i), TenantID: "t1", SessionID: "s2", EventName: "click"})
	}
	for i := 0; i < 3; i++ {
		db.Create(&model.Event{EventID: idFor("s3", i), TenantID: "t2", SessionID: "s3", EventName: "click"})
	}

	n, err := svc.Materialize(c.ID)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 member, got %d", n)
	}

	members, err := svc.ListMembers(c.ID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(members) != 1 || members[0] != "s1" {
		t.Fatalf("expected exactly [s1], got %v", members)
	}
}

func TestCohortServiceMaterializeIsIdempotentAndReplacesStaleMembers(t *testing.T) {
	db := newCohortTestDB(t)
	svc := &service.CohortService{DB: db}
	c, _ := svc.Create("t1", "power_clickers", "click", 1)

	db.Create(&model.Event{EventID: "e1", TenantID: "t1", SessionID: "s1", EventName: "click"})
	if _, err := svc.Materialize(c.ID); err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	members, _ := svc.ListMembers(c.ID)
	if len(members) != 1 || members[0] != "s1" {
		t.Fatalf("expected [s1] after first run, got %v", members)
	}

	// s1 no longer qualifies (deleted its events), s2 now does.
	db.Where("session_id = ?", "s1").Delete(&model.Event{})
	db.Create(&model.Event{EventID: "e2", TenantID: "t1", SessionID: "s2", EventName: "click"})

	if _, err := svc.Materialize(c.ID); err != nil {
		t.Fatalf("second materialize: %v", err)
	}
	members, _ = svc.ListMembers(c.ID)
	if len(members) != 1 || members[0] != "s2" {
		t.Fatalf("expected membership to be replaced with [s2], got %v", members)
	}
}

func idFor(session string, i int) string {
	return session + "-e" + string(rune('a'+i))
}
