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

func TestModelRegistryRejectsChangedArtifact(t *testing.T) {
	root := t.TempDir()
	artifact := []byte(`{"algorithm":"test"}`)
	registry := ModelRegistry{Root: root}
	manifest := NewManifest("buyer", "1", "buyer", "test", "v1", "model.json", "ghcr", artifact, time.Now())
	if err := registry.Store("global", "buyer", manifest, artifact); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "global", "buyer", "model.json"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Load("global", "buyer"); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestModelRegistryRejectsUnsafeAndIncompleteModels(t *testing.T) {
	registry := ModelRegistry{Root: t.TempDir()}
	artifact := []byte("model")
	valid := NewManifest("buyer", "1", "buyer", "nb", "v1", "model.json", "ghcr", artifact, time.Now())
	if err := registry.Store("../escape", "buyer", valid, artifact); err == nil {
		t.Fatal("unsafe scope accepted")
	}
	valid.Artifact = "../escape"
	if err := registry.Store("global", "buyer", valid, artifact); err == nil {
		t.Fatal("unsafe artifact path accepted")
	}
	valid = ModelManifest{Name: "buyer", Version: "1", Task: "buyer", Artifact: "model.json"}
	if err := registry.Store("global", "buyer", valid, artifact); err == nil {
		t.Fatal("incomplete manifest accepted")
	}
}
