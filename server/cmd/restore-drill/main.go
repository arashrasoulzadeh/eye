// restore-drill is a genuinely runnable disaster-recovery drill: given a
// snapshot produced by service.SnapshotService (a file under a local
// snapshot directory, Phase 11), it restores that snapshot into a fresh
// SQLite file and prints row counts per table so an operator can eyeball
// that the restore actually contains data, rather than this being
// documentation-only.
//
// Usage:
//
//	go run ./cmd/restore-drill --snapshot-dir=../snapshots --name=eye-snapshot-....db --dest=/tmp/restored.db
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func main() {
	snapshotDir := pflag.String("snapshot-dir", "./snapshots", "directory holding snapshots")
	name := pflag.String("name", "", "snapshot file name to restore (required)")
	dest := pflag.String("dest", "./restored.db", "path to restore the snapshot into")
	pflag.Parse()

	if *name == "" {
		fmt.Fprintln(os.Stderr, "--name is required")
		os.Exit(1)
	}

	store := &service.LocalFSSnapshotStore{Dir: *snapshotDir}
	ctx := context.Background()
	if err := service.Restore(ctx, store, *name, *dest); err != nil {
		fmt.Fprintf(os.Stderr, "restore failed: %v\n", err)
		os.Exit(1)
	}

	db, err := gorm.Open(sqlite.Open(*dest), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "open restored db failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("restored %s -> %s\n", *name, *dest)
	for _, t := range []struct {
		label string
		model any
	}{
		{"tenants", &model.Tenant{}},
		{"sessions", &model.Session{}},
		{"events", &model.Event{}},
	} {
		var count int64
		if err := db.Model(t.model).Count(&count).Error; err != nil {
			fmt.Fprintf(os.Stderr, "count %s failed: %v\n", t.label, err)
			continue
		}
		fmt.Printf("  %-10s %d rows\n", t.label, count)
	}
}
