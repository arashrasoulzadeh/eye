package test

import (
	"context"
	"testing"
	"time"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/service"
)

func TestRunCohortMaterializationJobRecomputesOnTick(t *testing.T) {
	db := newCohortTestDB(t)
	svc := &service.CohortService{DB: db}
	c, err := svc.Create("t1", "power_clickers", "click", 1)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go service.RunCohortMaterializationJob(ctx, svc, 10*time.Millisecond, nil)
	defer cancel()

	// No events yet: membership should stay empty across ticks.
	time.Sleep(30 * time.Millisecond)
	members, err := svc.ListMembers(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 0 {
		t.Fatalf("expected no members yet, got %v", members)
	}

	db.Create(&model.Event{EventID: "e1", TenantID: "t1", SessionID: "s1", EventName: "click"})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		members, err := svc.ListMembers(c.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(members) == 1 && members[0] == "s1" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("expected the background job to pick up the new event and add s1 to the cohort")
}
