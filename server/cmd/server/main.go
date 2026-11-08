package main

import (
	"log"
	"net/http"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/server/internal/config"
	"github.com/eye/eye/server/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil { log.Fatal(err) }
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil { log.Fatal(err) }
	repo := repository.New(db)
	if err := repo.Migrate(); err != nil { log.Fatal(err) }
	log.Println("server starting on", cfg.Addr())
	http.ListenAndServe(cfg.Addr(), nil)
}