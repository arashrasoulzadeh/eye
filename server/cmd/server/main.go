package main

import (
	"log/slog"
	"net/http"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/config"
	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

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
	if err != nil { logger.Error("config load failed", "err", err); os.Exit(1) }
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil { logger.Error("db open failed", "err", err); os.Exit(1) }
	repo := repository.New(db)
	if err := repo.Migrate(); err != nil { logger.Error("migrate failed", "err", err); os.Exit(1) }
	adminSvc := &service.AdminService{Repo: repo, Salt: cfg.APIKeySalt}
	svc := &service.IngestService{Repo: repo}
	mux := http.NewServeMux()
	adminAuth := func(h http.Handler) http.Handler { return handler.AdminAuth(cfg.AdminKey, h) }
	mux.Handle("/tenants", adminAuth(&handler.TenantsHandler{Svc: adminSvc}))
	mux.Handle("/tenants/", adminAuth(&handler.TenantsHandler{Svc: adminSvc}))
	mux.Handle("/schemas", adminAuth(&handler.SchemasHandler{Svc: adminSvc}))
	mux.Handle("/sessions", &handler.SessionsHandler{DB: db})
	mux.Handle("/events", &handler.EventsHandler{DB: db})
	qs := &service.QueryService{DB: db}
	mux.Handle("/query", &handler.QueryHandler{Svc: qs})
	ih := &handler.IngestHandler{Svc: svc}
	mux.Handle("/ingest", ih)
	mux.Handle("/metrics", &handler.MetricsHandler{DB: db})
	logger.Info("server starting", "addr", cfg.Addr())
	if err := http.ListenAndServe(cfg.Addr(), cors(mux)); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}