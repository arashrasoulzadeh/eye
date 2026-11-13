package service

import (
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserIntelligenceRefreshTagsBuyerAndBot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Event{}, &model.UserIntelligence{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	db.Create(&model.Session{ID: "s1", TenantID: "t1", IP: "1.2.3.4", UserAgent: "Googlebot", StartedAt: now, LastSeen: now})
	db.Create(&model.Event{EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "purchase", ClientTS: now.UnixMilli()})
	if _, err := (&UserIntelligenceService{DB: db}).Refresh("t1", now); err != nil {
		t.Fatal(err)
	}
	rows, err := (&UserIntelligenceService{DB: db}).List("t1", 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	seen := map[string]bool{}
	for _, tag := range rows[0].Tags {
		seen[tag] = true
	}
	if !seen["buyer"] || !seen["bot"] {
		t.Fatalf("tags=%v", rows[0].Tags)
	}
}

func TestTrainingReadinessRequiresBalancedLabels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.UserIntelligenceLabel{}); err != nil {
		t.Fatal(err)
	}
	svc := &UserIntelligenceService{DB: db}
	for i := 0; i < 5; i++ {
		if err := svc.AddLabel("t1", UserFingerprint("ip", string(rune('a'+i))), "buyer", "test"); err != nil {
			t.Fatal(err)
		}
	}
	ready, err := svc.TrainingReadiness("t1", "buyer")
	if err != nil {
		t.Fatal(err)
	}
	if ready["ready"] != false || ready["positive"] != int64(5) {
		t.Fatalf("unexpected readiness: %#v", ready)
	}
}
