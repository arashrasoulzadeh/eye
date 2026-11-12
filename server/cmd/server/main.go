package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	eyev1 "github.com/arashrasoulzadeh/eye/gen/go/eyev1"
	"github.com/arashrasoulzadeh/eye/internal/config"
	"github.com/arashrasoulzadeh/eye/internal/handler"
	"github.com/arashrasoulzadeh/eye/internal/metrics"
	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
	"github.com/arashrasoulzadeh/eye/internal/service"
	"github.com/arashrasoulzadeh/eye/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func cors(origins string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, origin := range strings.Split(origins, ",") {
		if value := strings.TrimSpace(origin); value != "" {
			allowed[value] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowed["*"] || allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
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
	if err := cfg.ValidateForProduction(); err != nil {
		logger.Error("invalid production configuration", "err", err)
		os.Exit(1)
	}
	db, err := storage.OpenDatabase(cfg)
	if err != nil {
		logger.Error("db open failed", "err", err)
		os.Exit(1)
	}
	repo := repository.New(db)
	if err := repo.Migrate(); err != nil {
		logger.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	if cfg.DBDriver == "postgres" || cfg.DBDriver == "postgresql" || cfg.DBDriver == "timescale" {
		if err := storage.ApplyPostgresFeatures(db); err != nil {
			logger.Error("postgres migrations failed", "err", err)
			os.Exit(1)
		}
		if cfg.EnablePostgresRLS {
			if err := storage.ApplyPostgresRLS(db); err != nil {
				logger.Error("postgres RLS migration failed", "err", err)
				os.Exit(1)
			}
		}
	}
	adminSvc := &service.AdminService{Repo: repo, Salt: cfg.APIKeySalt, APIKeyTTL: time.Duration(cfg.APIKeyTTLHours) * time.Hour}
	expSvc := &service.ExperimentService{DB: db}
	// Phase 14: per-tenant rate tiers. Only the synchronous /ingest path
	// (IngestService) is tier-limited; the async /ingest/stream path
	// (StreamIngestPipeline) has its own separate ingestion logic and is
	// not wired to this limiter yet — an honest, documented scope limit.
	tenantLimiter := service.NewTenantRateLimiter()
	quotaLimiter := &service.TenantQuotaLimiter{DB: db}
	// Phase 14: real-time per-tenant event counters, streamed live over
	// gRPC (LiveCounters below). liveCounters is bumped synchronously by
	// IngestService on every accepted event.
	liveCounters := service.NewLiveCounterHub()
	// Phase 14: real-time per-session updates (EventCount, LastSeen),
	// streamed live via SSE. liveSessions is updated by both IngestService
	// (sync path) and StreamIngestPipeline (async path) on every accepted event.
	liveSessions := service.NewLiveSessionHub()
	piiFields := strings.Split(cfg.PIIFields, ",")
	svc := &service.IngestService{Repo: repo, ExpSvc: expSvc, TierLimiter: tenantLimiter, QuotaLimiter: quotaLimiter, Counters: liveCounters, Sessions: liveSessions, PIIFields: piiFields, DetectPII: cfg.PIIDetection, SchemaMode: cfg.SchemaMode}
	if cfg.DBDriver == "postgres" || cfg.DBDriver == "postgresql" || cfg.DBDriver == "timescale" {
		svc.SetTenantContext = storage.SetTenantContext
	}
	mux := http.NewServeMux()
	mux.Handle("/healthz", handler.HealthHandler{})
	// Phase 14: audit log for admin actions. Audit must sit INSIDE AdminAuth
	// so a request that fails authentication is never logged as an action —
	// only requests that actually passed admin auth reach it. See
	// handler.Audit's doc comment for the single-shared-admin-key caveat.
	adminAuth := func(h http.Handler) http.Handler { return handler.AdminAuth(cfg.AdminKey, handler.Audit(db, h)) }
	adminUserRole := func(email string) (string, bool) {
		var user model.AdminUser
		if err := db.Where("email = ? AND status = ?", email, "active").First(&user).Error; err != nil {
			return "", false
		}
		return user.Role, true
	}
	adminUserAuth := func(h http.Handler) http.Handler {
		return handler.AdminRoleAuth(cfg.AdminKey, adminUserRole, []string{"owner", "admin"}, handler.Audit(db, h))
	}
	mux.Handle("/tenants", adminAuth(&handler.TenantsHandler{Svc: adminSvc}))
	mux.Handle("/tenants/", adminAuth(&handler.TenantsHandler{Svc: adminSvc}))
	mux.Handle("/tenants/data/", adminAuth(&handler.TenantDataHandler{DB: db}))
	mux.Handle("/tenant-usage/", adminAuth(&handler.UsageHandler{DB: db, Svc: &service.UsageService{DB: db}}))
	mux.Handle("/schemas", adminAuth(&handler.SchemasHandler{Svc: adminSvc}))
	mux.Handle("/schemas/", adminAuth(&handler.SchemasHandler{Svc: adminSvc}))
	// /sessions accepts the admin key OR a tenant's own API key for POST
	// (session creation) — a public CDN SDK running in the browser must be
	// able to call this with only its tenant key, never the admin key.
	setTenantContext := storage.SetTenantContext
	if cfg.DBDriver != "postgres" && cfg.DBDriver != "postgresql" && cfg.DBDriver != "timescale" {
		setTenantContext = nil
	}
	mux.Handle("/sessions", handler.Audit(db, handler.TenantOrAdminAuth(cfg.AdminKey, adminSvc.VerifyAPIKey, &handler.SessionsHandler{DB: db, SetTenantContext: setTenantContext})))
	complianceSvc := &service.ComplianceService{DB: db}
	mux.Handle("/compliance/erase", adminAuth(&handler.ComplianceHandler{Svc: complianceSvc}))
	eventsHandler := adminAuth(&handler.EventsHandler{DB: db})
	mux.Handle("/events", eventsHandler)
	mux.Handle("/events/", eventsHandler)
	qs := &service.QueryService{DB: db}
	if cfg.DBDriver == "postgres" || cfg.DBDriver == "postgresql" || cfg.DBDriver == "timescale" {
		qs.SetTenantContext = storage.SetTenantContext
	}
	mux.Handle("/query", adminAuth(&handler.QueryHandler{Svc: qs}))
	analyticsSvc := &service.AnalyticsService{DB: db}
	mux.Handle("/analytics/funnel", adminAuth(&handler.FunnelHandler{Svc: analyticsSvc}))
	mux.Handle("/analytics/funnel/suggested", adminAuth(&handler.SuggestedFunnelHandler{Svc: analyticsSvc}))
	mux.Handle("/analytics/retention", adminAuth(&handler.RetentionHandler{Svc: analyticsSvc}))
	mux.Handle("/analytics/views", adminAuth(&handler.AnalyticsViewsHandler{DB: db}))
	mux.Handle("/analytics/session-events", adminAuth(&handler.SessionEventsHandler{Svc: analyticsSvc}))
	mux.Handle("/analytics/devices", adminAuth(&handler.DeviceBreakdownHandler{Svc: analyticsSvc}))
	mux.Handle("/analytics/top-n", adminAuth(&handler.TopNHandler{Svc: analyticsSvc}))
	mux.Handle("/analytics/sliding-window", adminAuth(&handler.SlidingWindowHandler{Svc: analyticsSvc}))
	mux.Handle("/analytics/distinct-sessions", adminAuth(&handler.DistinctSessionsHandler{Svc: analyticsSvc}))
	mux.Handle("/analytics/percentiles", adminAuth(&handler.PercentilesHandler{Svc: analyticsSvc}))
	mux.Handle("/analytics/geography", adminAuth(&handler.GeoBreakdownHandler{Svc: analyticsSvc}))
	mux.Handle("/export/events", adminAuth(&handler.ExportHandler{Svc: qs}))
	cohortSvc := &service.CohortService{DB: db}
	mux.Handle("/cohorts", adminAuth(&handler.CohortsHandler{Svc: cohortSvc}))
	mux.Handle("/cohorts/", adminAuth(&handler.CohortsHandler{Svc: cohortSvc}))
	aggSvc := &service.AggregatorService{DB: db}
	mux.Handle("/aggregations", adminAuth(&handler.AggregationsHandler{Svc: aggSvc}))
	mux.Handle("/aggregations/", adminAuth(&handler.AggregationsHandler{Svc: aggSvc}))
	mux.Handle("/aggregations/export", adminAuth(&handler.AggregationExportHandler{Svc: aggSvc, Client: &service.HTTPWebhookClient{Client: http.DefaultClient}}))
	mux.Handle("/experiments", adminAuth(&handler.ExperimentsHandler{Svc: expSvc}))
	mux.Handle("/experiments/", adminAuth(&handler.ExperimentsHandler{Svc: expSvc}))

	// Phase 14: per-tenant feature flags. Nothing in this codebase gates
	// behavior on one yet — the deliverable is the capability to define
	// and check flags per tenant.
	flagSvc := &service.FeatureFlagService{DB: db}
	mux.Handle("/feature-flags", adminAuth(&handler.FeatureFlagsHandler{Svc: flagSvc}))
	// Phase 8: adaptive sampling under load. The controller's rate is driven
	// by the Phase 7 stream's queue depth (see the ticker below) and shared
	// across both ingest paths so a backed-up async queue also throttles the
	// synchronous one, since they feed the same downstream storage.
	sampler := service.NewSamplingController()

	// Phase 12: retention + the ingest rate limit become genuinely
	// DB-backed and live-updatable via PUT /settings (see settings.go) —
	// not just env-file values read once at startup.
	settingsSvc := &service.SettingsService{Repo: repo, DefaultRetentionHours: cfg.RetentionHours, DefaultRateLimitPerSecond: 50}
	mux.Handle("/settings", adminAuth(&handler.SettingsHandler{Svc: settingsSvc}))
	liveLimiter := &service.LiveRateLimiter{Provider: settingsSvc}

	ih := &handler.IngestHandler{Svc: svc, Sampler: sampler, Limiter: liveLimiter}
	lookupSessionTenant := func(sessionID string) (string, bool) {
		var session model.Session
		if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
			return "", false
		}
		return session.TenantID, true
	}
	mux.Handle("/ingest", handler.TenantIngestAuth(cfg.AdminKey, adminSvc.VerifyAPIKey, lookupSessionTenant, ih))

	// Phase 7: an async ingest path backed by an in-process stream
	// (ChannelStream — see internal/service/stream.go for why this is an
	// honest dev substitute for Kafka/Redpanda, not the real thing) whose
	// consumer inserts events and maintains materialized aggregate counts.
	var stream service.StreamProducer
	var streamConsumer service.StreamConsumer
	queueDepth := func() int { return 0 }
	if cfg.BrokerURL != "" {
		redisStream, err := service.NewRedisStream(cfg.BrokerURL, cfg.BrokerStream, cfg.BrokerGroup, cfg.BrokerConsumer)
		if err != nil {
			logger.Error("broker setup failed", "err", err)
			os.Exit(1)
		}
		stream, streamConsumer = redisStream, redisStream
	} else {
		channelStream := service.NewChannelStream(256)
		stream, streamConsumer = channelStream, channelStream
		queueDepth = channelStream.QueueDepth
	}
	streamPipeline := &service.StreamIngestPipeline{Repo: repo, DB: db, QueueDB: db, Producer: stream, Sessions: liveSessions, TierLimiter: tenantLimiter, QuotaLimiter: quotaLimiter, PIIFields: piiFields, DetectPII: cfg.PIIDetection, SchemaMode: cfg.SchemaMode}
	if cfg.DBDriver == "postgres" || cfg.DBDriver == "postgresql" || cfg.DBDriver == "timescale" {
		streamPipeline.SetTenantContext = storage.SetTenantContext
	}
	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	streamConsumer.Start(streamCtx, streamPipeline.Consume)
	if err := streamPipeline.ReplayPending(stream); err != nil {
		logger.Error("pending stream replay failed", "err", err)
	}
	mux.Handle("/ingest/stream", handler.TenantIngestAuth(cfg.AdminKey, adminSvc.VerifyAPIKey, lookupSessionTenant, &handler.IngestHandler{Svc: streamPipeline, Sampler: sampler, Limiter: liveLimiter}))
	mux.Handle("/aggregate-counts", adminAuth(&handler.AggregateCountsHandler{DB: db}))

	mux.Handle("/metrics", adminAuth(&handler.MetricsHandler{DB: db}))
	mux.Handle("/slo", adminAuth(handler.SLOHandler{}))
	// Dashboard cache: serves pre-computed metrics updated by a background job.
	dashboardCacheSvc := &service.DashboardCacheService{DB: db}
	mux.Handle("/dashboard-cache", adminAuth(&handler.DashboardCacheHandler{Svc: dashboardCacheSvc}))
	questionSvc := &service.QuestionService{DB: db}
	questionsHandler := adminAuth(&handler.QuestionsHandler{Svc: questionSvc})
	mux.Handle("/questions", questionsHandler)
	mux.Handle("/questions/", questionsHandler)
	mux.Handle("/event-options", adminAuth(&handler.EventOptionsHandler{DB: db}))
	// Phase 14: SSE bridge for the admin panel, since a browser can't speak
	// gRPC directly — wraps the same liveCounters hub the LiveCounters gRPC
	// stream above serves.
	mux.Handle("/live-counters", adminAuth(&handler.LiveCountersHandler{Hub: liveCounters}))
	// Phase 14: SSE bridge for live per-session updates (EventCount, LastSeen).
	mux.Handle("/live-session", adminAuth(&handler.LiveSessionHandler{Hub: liveSessions}))
	// Phase 14: SSE bridge for live per-session events.
	mux.Handle("/live-events", adminAuth(&handler.LiveEventsHandler{Hub: liveSessions}))

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
				depth := queueDepth()
				metrics.StreamQueueDepth.Set(float64(depth))
				metrics.IngestSampleRate.Set(sampler.UpdateRate(depth))
			}
		}
	}()

	retentionCtx, cancelRetention := context.WithCancel(context.Background())
	defer cancelRetention()
	go storage.RunRetentionJob(retentionCtx, storage.NewTimescaleRepo(db), settingsSvc.CurrentRetention, time.Hour, logger)

	cohortCtx, cancelCohort := context.WithCancel(context.Background())
	defer cancelCohort()
	go service.RunCohortMaterializationJob(cohortCtx, cohortSvc, 5*time.Minute, logger)

	aggCtx, cancelAgg := context.WithCancel(context.Background())
	defer cancelAgg()
	go service.RunAggregationJob(aggCtx, aggSvc, 5*time.Minute, logger)

	alertSvc := &service.AlertService{DB: db, DeadLetterNotifyURL: cfg.DeadLetterNotifyURL}
	if cfg.SMTPHost != "" {
		alertSvc.EmailClient = &service.SMTPWebhookClient{Host: cfg.SMTPHost, Port: cfg.SMTPPort, From: cfg.SMTPFrom, User: cfg.SMTPUser, Password: cfg.SMTPPassword}
	}
	mux.Handle("/alerts", adminAuth(&handler.AlertsHandler{Svc: alertSvc}))
	mux.Handle("/alerts/", adminAuth(&handler.AlertsHandler{Svc: alertSvc}))
	alertCtx, cancelAlert := context.WithCancel(context.Background())
	defer cancelAlert()
	go service.RunAlertJob(alertCtx, alertSvc, &service.HTTPWebhookClient{}, time.Minute, logger)

	// Phase 14: webhook delivery retries + dead-letter queue (see
	// DispatchWithRetry, now used by RunAlertJob above) and a listing
	// endpoint for deliveries that exhausted every retry.
	mux.Handle("/webhooks/dead-letter", adminAuth(&handler.WebhookDeadLetterHandler{Svc: &service.WebhookDeadLetterService{DB: db}}))

	// Phase 14: event replay — re-ingests a tenant's historical events for a
	// time range through the real ingest path (see service.ReplayService's
	// doc comment for the additive, non-duplicating design).
	mux.Handle("/replay", adminAuth(&handler.ReplayHandler{Svc: &service.ReplayService{DB: db, Ingest: svc}}))

	// Phase 14: audit log listing (writes happen via the Audit middleware
	// wrapping adminAuth above).
	mux.Handle("/audit-log", adminAuth(&handler.AuditLogHandler{DB: db}))
	mux.Handle("/admin-users", adminUserAuth(&handler.AdminUsersHandler{DB: db}))
	mux.Handle("/admin-users/", adminUserAuth(&handler.AdminUsersHandler{DB: db}))
	mux.Handle("/admin-users/accept", &handler.AdminInvitationHandler{DB: db})
	mux.Handle("/admin-teams", adminUserAuth(&handler.AdminTeamsHandler{DB: db}))
	mux.Handle("/admin-teams/", adminUserAuth(&handler.AdminTeamsHandler{DB: db}))

	// Phase 8: basic statistical (z-score over a rolling window) anomaly
	// detection, sampled from the same materialized AggregateCount read
	// model as the alert job — see anomaly.go's doc comments for why this is
	// a named simple method, not "ML".
	if err := db.AutoMigrate(&service.MetricSample{}, &model.AnalyticsView{}); err != nil {
		logger.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	anomalySvc := service.NewAnomalyService(db)
	mux.Handle("/anomalies", adminAuth(&handler.AnomaliesHandler{Svc: anomalySvc}))
	anomalyCtx, cancelAnomaly := context.WithCancel(context.Background())
	defer cancelAnomaly()
	go service.RunAnomalySamplingJob(anomalyCtx, anomalySvc, 5*time.Minute, logger)

	// Phase 11: periodic local-filesystem snapshots (see snapshot.go for why
	// this is a local substitute, not real S3).
	snapshotSvc := &service.SnapshotService{DB: db, DBPath: cfg.DBPath, Store: &service.LocalFSSnapshotStore{Dir: cfg.SnapshotDir}}
	snapshotCtx, cancelSnapshot := context.WithCancel(context.Background())
	defer cancelSnapshot()
	go service.RunSnapshotJob(snapshotCtx, snapshotSvc, time.Hour, func(err error) {
		logger.Error("snapshot job failed", "err", err)
	})

	// Dashboard cache: periodic refresh of pre-computed dashboard metrics.
	// Default 1 minute, configurable via settings.
	dashboardCacheCtx, cancelDashboardCache := context.WithCancel(context.Background())
	defer cancelDashboardCache()
	go service.RunDashboardCacheJob(dashboardCacheCtx, dashboardCacheSvc, time.Minute, logger)

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
	eyev1.RegisterCollectorServer(gs, &service.QueryStreamServer{Svc: qs, PageSize: 100, Hub: liveCounters})
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
	origins := cfg.CORSOrigins
	if origins == "" && !strings.EqualFold(os.Getenv("EYE_ENV"), "production") {
		origins = "*"
	}
	if err := http.ListenAndServe(cfg.Addr(), handler.Observability(cors(origins, mux))); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
