package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gorm.io/gorm"
)

// SnapshotStore is a swappable destination for database snapshots — the
// same honest-substitute seam used for Phase 7's StreamProducer/Consumer.
// There is no real AWS/S3 access in this dev environment (CLAUDE.md: SQLite
// for dev, no Docker, no cloud infra), so only a local-filesystem
// implementation is provided here and verified. A real S3-backed
// implementation (Put -> PutObject, List -> ListObjectsV2, Get ->
// GetObject) can satisfy this same interface later without touching
// SnapshotService or its callers; it was not attempted because it would be
// unverifiable without real AWS credentials.
type SnapshotStore interface {
	Put(ctx context.Context, name string, r io.Reader) error
	List(ctx context.Context) ([]string, error)
	Get(ctx context.Context, name string) (io.ReadCloser, error)
}

// LocalFSSnapshotStore stores snapshots as files in Dir. Put writes to a
// temp file in the same directory and renames it into place, so a reader
// never observes a partially-written snapshot (os.Rename is atomic within
// one filesystem/directory).
type LocalFSSnapshotStore struct{ Dir string }

func (s *LocalFSSnapshotStore) Put(ctx context.Context, name string, r io.Reader) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Dir, ".snapshot-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, filepath.Join(s.Dir, name))
}

func (s *LocalFSSnapshotStore) List(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) == ".tmp" {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

func (s *LocalFSSnapshotStore) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.Dir, name))
}

// SnapshotService produces point-in-time snapshots of the dev SQLite
// database. DBPath must be the on-disk file GORM has open as DB — a
// ":memory:" database has nothing to snapshot via VACUUM INTO.
type SnapshotService struct {
	DB     *gorm.DB
	DBPath string
	Store  SnapshotStore
}

// TakeSnapshot produces a single, internally-consistent copy of the
// database and hands it to Store. It uses SQLite's VACUUM INTO, which
// writes a complete, consistent copy to a new file without holding a
// long-lived write lock on the live database — the standard way to get a
// safe snapshot of a SQLite file that may still be receiving writes.
// The VACUUM INTO target must not already exist, so it's written to a
// fresh temp file first; Store.Put (for LocalFSSnapshotStore) does its own
// atomic rename on top of that.
func (s *SnapshotService) TakeSnapshot(ctx context.Context) (string, error) {
	tmpFile, err := os.CreateTemp("", "eye-snapshot-*.db")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	os.Remove(tmpPath) // VACUUM INTO requires the target not to exist yet
	defer os.Remove(tmpPath)

	if err := s.DB.Exec("VACUUM INTO ?", tmpPath).Error; err != nil {
		return "", fmt.Errorf("vacuum into: %w", err)
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	name := fmt.Sprintf("eye-snapshot-%s.db", time.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := s.Store.Put(ctx, name, f); err != nil {
		return "", err
	}
	return name, nil
}

// Restore copies a named snapshot out of Store and into destPath, ready to
// be opened as its own SQLite database (e.g. via gorm.Open(sqlite.Open(destPath))).
// It does not open or verify the restored file itself — that's the caller's
// job (see server/cmd/restore-drill for a runnable example) — so this stays
// a pure, storage-agnostic copy.
func Restore(ctx context.Context, store SnapshotStore, name, destPath string) error {
	r, err := store.Get(ctx, name)
	if err != nil {
		return err
	}
	defer r.Close()

	if dir := filepath.Dir(destPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

// RunSnapshotJob periodically takes a snapshot until ctx is cancelled, the
// same ticker-based pattern as RunCohortMaterializationJob/RunAggregationJob/
// RunAlertJob.
func RunSnapshotJob(ctx context.Context, svc *SnapshotService, interval time.Duration, onError func(error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := svc.TakeSnapshot(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}
