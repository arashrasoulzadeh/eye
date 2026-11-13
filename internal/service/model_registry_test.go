package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestModelRegistryResolvesTenantBeforeGlobal(t *testing.T) {
	root := t.TempDir()
	artifact := []byte(`{"algorithm":"test"}`)
	for _, item := range []struct{ scope, version string }{{"global", "1"}, {filepath.Join("tenants", "t1"), "2"}} {
		dir := filepath.Join(root, item.scope, "buyer")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		manifest := NewManifest("buyer", item.version, "buyer", "test", "v1", "model.json", "ghcr", artifact, time.Now())
		encoded, _ := json.Marshal(manifest)
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), encoded, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "model.json"), artifact, 0644); err != nil {
			t.Fatal(err)
		}
	}
	manifest, scope, err := (ModelRegistry{Root: root}).Resolve("t1", "buyer")
	if err != nil || scope != "tenant" || manifest.Version != "2" {
		t.Fatalf("manifest=%+v scope=%s err=%v", manifest, scope, err)
	}
}
