package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eye/eye/internal/config"
	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/repository"
	"github.com/eye/eye/internal/service"
	"github.com/eye/eye/internal/storage"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,x-api-key,x-admin-key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		logger.Error("db open failed", "err", err)
		os.Exit(1)
	}
	repo := repository.New(db)
	if err := repo.Migrate(); err != nil {
		logger.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	adminSvc := &service.AdminService{Repo: repo, Salt: cfg.APIKeySalt}
	expSvc := &service.ExperimentService{DB: db}
	svc := &service.IngestService{Repo: repo, ExpSvc: expSvc}
	mux := http.NewServeMux()
	adminAuth := func(h http.Handler) http.Handler { return handler.AdminAuth(cfg.AdminKey, h) }
	mux.Handle("/tenants", adminAuth(&handler.TenantsHandler{Svc: adminSvc}))
	mux.Handle("/tenants/", adminAuth(&handler.TenantsHandler{Svc: adminSvc}))
	mux.Handle("/schemas", adminAuth(&handler.SchemasHandler{Svc: adminSvc}))
	mux.Handle("/sessions", adminAuth(&handler.SessionsHandler{DB: db}))
	mux.Handle("/events", adminAuth(&handler.EventsHandler{DB: db}))
	qs := &service.QueryService{DB: db}
	mux.Handle("/query", adminAuth(&handler.QueryHandler{Svc: qs}))
	cohortSvc := &service.CohortService{DB: db}
	mux.Handle("/cohorts", adminAuth(&handler.CohortsHandler{Svc: cohortSvc}))
	mux.Handle("/cohorts/", adminAuth(&handler.CohortsHandler{Svc: cohortSvc}))
	aggSvc := &service.AggregatorService{DB: db}
	mux.Handle("/aggregations", adminAuth(&handler.AggregationsHandler{Svc: aggSvc}))
	mux.Handle("/aggregations/", adminAuth(&handler.AggregationsHandler{Svc: aggSvc}))
	mux.Handle("/experiments", adminAuth(&handler.ExperimentsHandler{Svc: expSvc}))
	mux.Handle("/experiments/", adminAuth(&handler.ExperimentsHandler{Svc: expSvc}))
	ih := &handler.IngestHandler{Svc: svc}
	mux.Handle("/ingest", ih)

	// Phase 7: an async ingest path backed by an in-process stream
	// (ChannelStream — see internal/service/stream.go for why this is an
	// honest dev substitute for Kafka/Redpanda, not the real thing) whose
	// consumer inserts events and maintains materialized aggregate counts.
	stream := service.NewChannelStream(256)
	streamPipeline := &service.StreamIngestPipeline{Repo: repo, DB: db, Producer: stream}
	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	stream.Start(streamCtx, streamPipeline.Consume)
	mux.Handle("/ingest/stream", &handler.IngestHandler{Svc: streamPipeline})
	mux.Handle("/aggregate-counts", adminAuth(&handler.AggregateCountsHandler{DB: db}))

	mux.Handle("/metrics", adminAuth(&handler.MetricsHandler{DB: db}))

	retentionCtx, cancelRetention := context.WithCancel(context.Background())
	defer cancelRetention()
	go storage.RunRetentionJob(retentionCtx, storage.NewTimescaleRepo(db), time.Duration(cfg.RetentionHours)*time.Hour, time.Hour, logger)

	cohortCtx, cancelCohort := context.WithCancel(context.Background())
	defer cancelCohort()
	go service.RunCohortMaterializationJob(cohortCtx, cohortSvc, 5*time.Minute, logger)

	aggCtx, cancelAgg := context.WithCancel(context.Background())
	defer cancelAgg()
	go service.RunAggregationJob(aggCtx, aggSvc, 5*time.Minute, logger)

	logger.Info("server starting", "addr", cfg.Addr())
	if err := http.ListenAndServe(cfg.Addr(), cors(mux)); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
