package service

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func newTestDB_feature_flag_test(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.FeatureFlag{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestFeatureFlagService(t *testing.T) {
	db := newTestDB_feature_flag_test(t)
	svc := &FeatureFlagService{DB: db}

	// Test Set and Get
	svc.Set("t1", "new-ui", true)
	enabled := svc.IsEnabled("t1", "new-ui")
	if !enabled {
		t.Error("expected new-ui to be enabled")
	}

	// Test update
	svc.Set("t1", "new-ui", false)
	enabled = svc.IsEnabled("t1", "new-ui")
	if enabled {
		t.Error("expected new-ui to be disabled")
	}

	// Test List
	flags, err := svc.List("t1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, f := range flags {
		if f.Key == "new-ui" {
			found = true
			if f.Enabled {
				t.Error("expected disabled in list")
			}
		}
	}
	if !found {
		t.Error("expected new-ui in list")
	}

	// Test Delete
	svc.Delete("t1", "new-ui")
	enabled = svc.IsEnabled("t1", "new-ui")
	if enabled {
		t.Error("expected deleted flag to be disabled")
	}
}

func TestFeatureFlagListPageFiltersAndOrders(t *testing.T) {
	db := newTestDB_feature_flag_test(t)
	db.Create(&model.FeatureFlag{ID: "one", TenantID: "t1", Key: "checkout.v1", UpdatedAt: 1})
	db.Create(&model.FeatureFlag{ID: "two", TenantID: "t1", Key: "checkout.v2", UpdatedAt: 2})
	items, total, err := (&FeatureFlagService{DB: db}).ListPage("t1", "v2", "desc", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != "two" {
		t.Fatalf("items=%+v total=%d", items, total)
	}
}
