package main

import (
	"log"
	"net/http"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/config"
	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
)

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,x-api-key")
		if r.Method == http.MethodOptions { w.WriteHeader(204); return }
		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg, err := config.Load()
	if err != nil { log.Fatal(err) }
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil { log.Fatal(err) }
	repo := repository.New(db)
	if err := repo.Migrate(); err != nil { log.Fatal(err) }
	adminSvc := &service.AdminService{Repo: repo}
	svc := &service.IngestService{Repo: repo}
	mux := http.NewServeMux()
	mux.Handle("/tenants", &handler.TenantsHandler{Svc: adminSvc})
	ih := &handler.IngestHandler{Svc: svc}
	mux.Handle("/ingest", ih)
	mux.Handle("/metrics", &handler.MetricsHandler{DB: db})
	log.Println("server starting on", cfg.Addr())
	http.ListenAndServe(cfg.Addr(), cors(mux))
}