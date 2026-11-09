package test

import (
	"testing"

	"github.com/eye/eye/internal/service"
)

func TestFeatureFlagServiceCRUDAndLookup(t *testing.T) {
	repo := newTierTestRepo(t)
	svc := &service.FeatureFlagService{DB: repo.DB}

	if svc.IsEnabled("t1", "new-ui") {
		t.Error("undefined flag should default to false")
	}

	if _, err := svc.Set("t1", "new-ui", true); err != nil {
		t.Fatal(err)
	}
	if !svc.IsEnabled("t1", "new-ui") {
		t.Error("expected enabled flag to report true")
	}

	if _, err := svc.Set("t1", "new-ui", false); err != nil {
		t.Fatal(err)
	}
	if svc.IsEnabled("t1", "new-ui") {
		t.Error("expected explicitly-disabled flag to report false")
	}

	if _, err := svc.Set("t1", "beta", true); err != nil {
		t.Fatal(err)
	}
	list, err := svc.List("t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 flags for t1, got %d", len(list))
	}

	// Flags are scoped per tenant.
	if svc.IsEnabled("t2", "new-ui") {
		t.Error("flag should not leak across tenants")
	}

	if err := svc.Delete("t1", "beta"); err != nil {
		t.Fatal(err)
	}
	list, _ = svc.List("t1")
	if len(list) != 1 {
		t.Errorf("expected 1 flag after delete, got %d", len(list))
	}

	if err := svc.Delete("t1", "does-not-exist"); err == nil {
		t.Error("expected error deleting a flag that doesn't exist")
	}
}
