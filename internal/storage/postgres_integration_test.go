package storage

import (
	"os"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/config"
	"github.com/arashrasoulzadeh/eye/internal/repository"
)

func TestPostgresProductionMigrations(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	db, err := OpenDatabase(&config.Config{DBDriver: "postgres", DatabaseURL: url})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.New(db).Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPostgresFeatures(db); err != nil {
		t.Fatal(err)
	}
}
