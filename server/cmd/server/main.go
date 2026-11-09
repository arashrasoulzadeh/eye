package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	eyev1 "github.com/eye/eye/gen/go/eyev1"
	"github.com/eye/eye/internal/config"
	"github.com/eye/eye/internal/handler"
	"github.com/eye/eye/internal/metrics"
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

	// Phase 9: real Prometheus exposition, separate from the JSON dashboard
	// stats already served at /metrics above (that path predates this work
	// and the admin UI depends on its JSON shape, so it's left alone rather
	// than overloading one path with two incompatible response formats).
	metrics.Init()
	mux.Handle("/metrics/prometheus", promhttp.Handler())
	queueCtx, cancelQueue := context.WithCancel(context.Background())
	defer cancelQueue()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-queueCtx.Done():
				return
			case <-ticker.C:
				metrics.StreamQueueDepth.Set(float64(stream.QueueDepth()))
			}
		}
	}()

	retentionCtx, cancelRetention := context.WithCancel(context.Background())
	defer cancelRetention()
	go storage.RunRetentionJob(retentionCtx, storage.NewTimescaleRepo(db), time.Duration(cfg.RetentionHours)*time.Hour, time.Hour, logger)

	cohortCtx, cancelCohort := context.WithCancel(context.Background())
	defer cancelCohort()
	go service.RunCohortMaterializationJob(cohortCtx, cohortSvc, 5*time.Minute, logger)

	aggCtx, cancelAgg := context.WithCancel(context.Background())
	defer cancelAgg()
	go service.RunAggregationJob(aggCtx, aggSvc, 5*time.Minute, logger)

	alertSvc := &service.AlertService{DB: db}
	mux.Handle("/alerts", adminAuth(&handler.AlertsHandler{Svc: alertSvc}))
	mux.Handle("/alerts/", adminAuth(&handler.AlertsHandler{Svc: alertSvc}))
	alertCtx, cancelAlert := context.WithCancel(context.Background())
	defer cancelAlert()
	go service.RunAlertJob(alertCtx, alertSvc, &service.HTTPWebhookClient{}, time.Minute, logger)

	// Phase 6: gRPC server-streaming QueryStream, backed by the same
	// DB-connected QueryService as the REST /query endpoint. This process
	// (not the separate in-memory server/cmd/collector) owns the DB, so
	// QueryStream is served here; CreateSession/Ingest/IngestStream remain
	// the collector binary's responsibility.
	grpcLis, err := net.Listen("tcp", ":50052")
	if err != nil {
		logger.Error("grpc listen failed", "err", err)
		os.Exit(1)
	}
	gs := grpc.NewServer()
	eyev1.RegisterCollectorServer(gs, &service.QueryStreamServer{Svc: qs, PageSize: 100})
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("eye.v1.Collector", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(gs, healthSrv)
	reflection.Register(gs)
	go func() {
		logger.Info("grpc query server listening", "addr", ":50052")
		if err := gs.Serve(grpcLis); err != nil {
			logger.Error("grpc serve failed", "err", err)
		}
	}()

	logger.Info("server starting", "addr", cfg.Addr())
	if err := http.ListenAndServe(cfg.Addr(), cors(mux)); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
