package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
)

func TestRepoMigrateAndSession(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	r := repository.New(db)
	if err := r.Migrate(); err != nil { t.Fatal(err) }
	s := &model.Session{ID: "s1", TenantID: "t1"}
	if err := r.CreateSession(s); err != nil { t.Fatal(err) }
	got, err := r.GetSession("s1")
	if err != nil || got.TenantID != "t1" { t.Fatalf("session mismatch: %v", err) }
}
