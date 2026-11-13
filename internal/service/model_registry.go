package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ModelManifest is the portable contract for global models downloaded from a
// GHCR artifact or later published to Hugging Face.
type ModelManifest struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Task          string `json:"task"`
	Algorithm     string `json:"algorithm"`
	FeatureSchema string `json:"feature_schema"`
	Artifact      string `json:"artifact"`
	SHA256        string `json:"sha256"`
	CreatedAt     string `json:"created_at"`
	Source        string `json:"source"`
}

type ModelRegistry struct{ Root string }

func (r ModelRegistry) manifestPath(scope, task string) string {
	return filepath.Join(r.Root, scope, task, "manifest.json")
}

func (r ModelRegistry) Load(scope, task string) (ModelManifest, error) {
	data, err := os.ReadFile(r.manifestPath(scope, task))
	if err != nil {
		return ModelManifest{}, err
	}
	var manifest ModelManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ModelManifest{}, fmt.Errorf("invalid model manifest: %w", err)
	}
	if manifest.Name == "" || manifest.Version == "" || manifest.Task == "" || manifest.Artifact == "" {
		return ModelManifest{}, fmt.Errorf("model manifest is missing required fields")
	}
	if manifest.Task != task {
		return ModelManifest{}, fmt.Errorf("model task %q does not match requested task %q", manifest.Task, task)
	}
	artifact := filepath.Join(filepath.Dir(r.manifestPath(scope, task)), manifest.Artifact)
	contents, err := os.ReadFile(artifact)
	if err != nil {
		return ModelManifest{}, fmt.Errorf("model artifact unavailable: %w", err)
	}
	sum := sha256.Sum256(contents)
	if manifest.SHA256 != "" && !strings.EqualFold(manifest.SHA256, hex.EncodeToString(sum[:])) {
		return ModelManifest{}, fmt.Errorf("model artifact checksum mismatch")
	}
	return manifest, nil
}

// Resolve prefers a tenant-pinned artifact and falls back to the global one.
func (r ModelRegistry) Resolve(tenantID, task string) (ModelManifest, string, error) {
	if tenantID != "" {
		if manifest, err := r.Load(filepath.Join("tenants", tenantID), task); err == nil {
			return manifest, "tenant", nil
		}
	}
	manifest, err := r.Load("global", task)
	if err != nil {
		return ModelManifest{}, "", err
	}
	return manifest, "global", nil
}

func NewManifest(name, version, task, algorithm, featureSchema, artifact, source string, contents []byte, now time.Time) ModelManifest {
	sum := sha256.Sum256(contents)
	return ModelManifest{Name: name, Version: version, Task: task, Algorithm: algorithm, FeatureSchema: featureSchema, Artifact: artifact, SHA256: hex.EncodeToString(sum[:]), CreatedAt: now.UTC().Format(time.RFC3339), Source: source}
}
