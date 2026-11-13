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

func (r ModelRegistry) Store(scope, task string, manifest ModelManifest, artifact []byte) error {
	if strings.TrimSpace(scope) == "" || strings.Contains(scope, "..") || filepath.IsAbs(scope) || strings.TrimSpace(task) == "" || strings.Contains(task, "..") {
		return fmt.Errorf("invalid model scope or task")
	}
	if manifest.Name == "" || manifest.Version == "" || manifest.Task != task || manifest.Algorithm == "" || manifest.FeatureSchema == "" || strings.TrimSpace(manifest.Source) == "" {
		return fmt.Errorf("model manifest is missing required fields")
	}
	if filepath.Base(manifest.Artifact) != manifest.Artifact || manifest.Artifact == "" || manifest.Artifact == "." || manifest.Artifact == ".." {
		return fmt.Errorf("invalid model artifact filename")
	}
	dir := filepath.Join(r.Root, scope, task)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	manifest.Artifact = filepath.Base(manifest.Artifact)
	sum := sha256.Sum256(artifact)
	manifest.SHA256 = hex.EncodeToString(sum[:])
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	artifactTemp, err := os.CreateTemp(dir, ".artifact-*")
	if err != nil {
		return err
	}
	tempArtifact := artifactTemp.Name()
	defer os.Remove(tempArtifact)
	if err := artifactTemp.Chmod(0644); err != nil {
		artifactTemp.Close()
		return err
	}
	if _, err := artifactTemp.Write(artifact); err != nil {
		artifactTemp.Close()
		return err
	}
	if err := artifactTemp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempArtifact, filepath.Join(dir, manifest.Artifact)); err != nil {
		return err
	}
	manifestTemp, err := os.CreateTemp(dir, ".manifest-*")
	if err != nil {
		return err
	}
	tempManifest := manifestTemp.Name()
	defer os.Remove(tempManifest)
	if err := manifestTemp.Chmod(0644); err != nil {
		manifestTemp.Close()
		return err
	}
	if _, err := manifestTemp.Write(encoded); err != nil {
		manifestTemp.Close()
		return err
	}
	if err := manifestTemp.Close(); err != nil {
		return err
	}
	return os.Rename(tempManifest, filepath.Join(dir, "manifest.json"))
}

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
	if filepath.Base(manifest.Artifact) != manifest.Artifact || manifest.Artifact == "." || manifest.Artifact == ".." {
		return ModelManifest{}, fmt.Errorf("model manifest contains unsafe artifact filename")
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

func (r ModelRegistry) Artifact(scope, task string) ([]byte, ModelManifest, error) {
	manifest, err := r.Load(scope, task)
	if err != nil {
		return nil, ModelManifest{}, err
	}
	data, err := os.ReadFile(filepath.Join(r.Root, scope, task, manifest.Artifact))
	return data, manifest, err
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
