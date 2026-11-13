package metrics

import (
	"sync/atomic"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

// IngestTotal counts ingest requests by outcome: "accepted"/"rejected" for
// events that made it through decoding (per-event, from the accepted/rejected
// tallies IngestHandler gets back from the Ingester), and "error" for
// requests that failed before any event-level accounting (e.g. bad JSON).
var IngestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ingest_events_total"}, []string{"status"})

// IngestLatency observes wall-clock time for one /ingest or /ingest/stream
// HTTP request (decode + Ingester.Ingest call), i.e. the ingest rate's
// counterpart "how long did it take" metric.
var IngestLatency = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "ingest_latency_seconds"})

// IngestErrorsTotal counts ingest requests that failed outright (couldn't be
// decoded/processed at all), labeled by a short reason so "errors" in the
// Phase 9 checklist has its own series distinct from a per-event "rejected"
// outcome (a rejected event, e.g. a duplicate, is not an error).
var IngestErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ingest_errors_total"}, []string{"reason"})

// StreamQueueDepth reports how many messages are currently buffered in the
// Phase 7 ChannelStream's channel — the in-process substitute for a real
// broker's queue depth (see internal/service/stream.go). Sampled
// periodically by a ticker in main.go rather than pushed synchronously,
// since nothing else needs a live push and a gauge is naturally a sampled
// value.
var StreamQueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{Name: "stream_queue_depth"})

// IngestSampleRate reports the Phase 8 SamplingController's current accept
// rate (1.0 normally, dropping as StreamQueueDepth rises) so the adaptive
// sampling behavior is directly observable, not just inferred from dropped
// counts.
var IngestSampleRate = prometheus.NewGauge(prometheus.GaugeOpts{Name: "ingest_sample_rate"})

var HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP request duration by method and status."}, []string{"method", "status"})
var HTTPRequestsTotal = prometheus.NewGauge(prometheus.GaugeOpts{Name: "http_requests_total", Help: "Total HTTP requests observed by the process."})
var HTTPErrorsTotal = prometheus.NewGauge(prometheus.GaugeOpts{Name: "http_errors_total", Help: "Total HTTP 5xx responses observed by the process."})
var StoredEvents = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_stored_events", Help: "Non-aggregated events currently stored."})
var StoredSessions = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_stored_sessions", Help: "Sessions currently stored."})
var StoredTenants = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_stored_tenants", Help: "Tenants currently stored."})
var StoredBuckets = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_stored_event_buckets", Help: "Durable event buckets currently stored."})
var IntelligenceProfiles = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_intelligence_profiles", Help: "User intelligence snapshots currently stored."})
var IntelligenceModels = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_intelligence_models", Help: "Persisted intelligence models."})
var IntelligenceTrainingRuns = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_intelligence_training_runs", Help: "Persisted intelligence training runs."})
var IntelligencePipelineActive = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_intelligence_pipeline_active", Help: "Intelligence pipelines currently queued or running."})
var IntelligencePipelineFailed = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_intelligence_pipeline_failed", Help: "Intelligence pipelines currently failed and requiring attention."})
var DatabaseRows = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "eye_database_rows", Help: "Rows currently stored in each Eye database table."}, []string{"table"})
var HTTPRouteDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_route_duration_seconds", Help: "HTTP request duration by normalized route, method, and status."}, []string{"route", "method", "status"})
var MLTrainingExamples = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_ml_training_examples", Help: "Examples used by the latest persisted ML training runs."})
var MLTrainingPositiveExamples = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_ml_training_positive_examples", Help: "Positive examples in persisted ML training runs."})
var MLTrainingNegativeExamples = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_ml_training_negative_examples", Help: "Negative examples in persisted ML training runs."})
var MLAnomalyScoreAverage = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_ml_anomaly_score_average", Help: "Average anomaly score across user intelligence snapshots."})
var MLPredictionConfidenceAverage = prometheus.NewGauge(prometheus.GaugeOpts{Name: "eye_ml_prediction_confidence_average", Help: "Average prediction confidence across user intelligence snapshots."})
var IntelligenceDriftScore = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "eye_intelligence_drift_score", Help: "Latest model drift score by tenant and label."}, []string{"tenant", "label"})
var IntelligenceCalibrationError = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "eye_intelligence_calibration_error", Help: "Latest model calibration error by tenant and label."}, []string{"tenant", "label"})
var IntelligenceHealth = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "eye_intelligence_health", Help: "Latest model health: 1 healthy, 0.5 warning, 0 drifted."}, []string{"tenant", "label"})

var httpRequests atomic.Int64
var httpErrors atomic.Int64

func RecordHTTPRequest(status int) {
	httpRequests.Add(1)
	if status >= 500 {
		httpErrors.Add(1)
	}
}

func RefreshDatabaseStats(db *gorm.DB) {
	var count int64
	if db.Model(&model.Event{}).Where("is_aggregated = ?", false).Count(&count).Error == nil {
		StoredEvents.Set(float64(count))
	}
	if db.Model(&model.Session{}).Count(&count).Error == nil {
		StoredSessions.Set(float64(count))
	}
	if db.Model(&model.Tenant{}).Count(&count).Error == nil {
		StoredTenants.Set(float64(count))
	}
	if db.Model(&model.EventBucket{}).Count(&count).Error == nil {
		StoredBuckets.Set(float64(count))
	}
	if db.Model(&model.UserIntelligence{}).Count(&count).Error == nil {
		IntelligenceProfiles.Set(float64(count))
	}
	if db.Model(&model.IntelligenceModel{}).Count(&count).Error == nil {
		IntelligenceModels.Set(float64(count))
	}
	if db.Model(&model.IntelligenceTrainingRun{}).Count(&count).Error == nil {
		IntelligenceTrainingRuns.Set(float64(count))
	}
	if db.Model(&model.IntelligencePipelineRun{}).Where("status IN ?", []string{"queued", "running"}).Count(&count).Error == nil {
		IntelligencePipelineActive.Set(float64(count))
	}
	if db.Model(&model.IntelligencePipelineRun{}).Where("status = ?", "failed").Count(&count).Error == nil {
		IntelligencePipelineFailed.Set(float64(count))
	}
	rows := []struct {
		name  string
		value any
	}{
		{"tenants", &model.Tenant{}}, {"feature_flags", &model.FeatureFlag{}}, {"sessions", &model.Session{}},
		{"events", &model.Event{}}, {"event_buckets", &model.EventBucket{}}, {"pending_stream_events", &model.PendingStreamEvent{}},
		{"aggregations", &model.Aggregation{}}, {"aggregate_counts", &model.AggregateCount{}}, {"schemas", &model.Schema{}},
		{"cohorts", &model.Cohort{}}, {"cohort_memberships", &model.CohortMembership{}}, {"experiments", &model.Experiment{}},
		{"experiment_assignments", &model.ExperimentAssignment{}}, {"alert_rules", &model.AlertRule{}}, {"runtime_settings", &model.RuntimeSettings{}},
		{"webhook_dead_letters", &model.WebhookDeadLetter{}}, {"audit_logs", &model.AuditLog{}}, {"admin_users", &model.AdminUser{}},
		{"admin_teams", &model.AdminTeam{}}, {"admin_team_members", &model.AdminTeamMember{}}, {"admin_sessions", &model.AdminSession{}},
		{"dashboard_caches", &model.DashboardCache{}}, {"questions", &model.Question{}}, {"analytics_views", &model.AnalyticsView{}},
		{"user_intelligence", &model.UserIntelligence{}}, {"user_intelligence_labels", &model.UserIntelligenceLabel{}},
		{"intelligence_training_runs", &model.IntelligenceTrainingRun{}}, {"intelligence_pipeline_runs", &model.IntelligencePipelineRun{}}, {"intelligence_models", &model.IntelligenceModel{}},
	}
	for _, row := range rows {
		if db.Model(row.value).Count(&count).Error == nil {
			DatabaseRows.WithLabelValues(row.name).Set(float64(count))
		}
	}
	var runs []model.IntelligenceTrainingRun
	if db.Find(&runs).Error == nil {
		var examples, positive, negative int
		for _, run := range runs {
			examples += run.Examples
			positive += run.Positive
			negative += run.Negative
		}
		MLTrainingExamples.Set(float64(examples))
		MLTrainingPositiveExamples.Set(float64(positive))
		MLTrainingNegativeExamples.Set(float64(negative))
	}
	var profiles []model.UserIntelligence
	if db.Find(&profiles).Error == nil && len(profiles) > 0 {
		var anomaly, confidence float64
		for _, profile := range profiles {
			anomaly += profile.AnomalyScore
			confidence += profile.Confidence
		}
		MLAnomalyScoreAverage.Set(anomaly / float64(len(profiles)))
		MLPredictionConfidenceAverage.Set(confidence / float64(len(profiles)))
	}
	total, errors := HTTPRequestSnapshot()
	HTTPRequestsTotal.Set(float64(total))
	HTTPErrorsTotal.Set(float64(errors))
}

func HTTPRequestSnapshot() (total, errors int64) { return httpRequests.Load(), httpErrors.Load() }

func Init() {
	prometheus.MustRegister(IngestTotal, IngestLatency, IngestErrorsTotal, StreamQueueDepth, IngestSampleRate, HTTPRequestDuration, HTTPRequestsTotal, HTTPErrorsTotal, StoredEvents, StoredSessions, StoredTenants, StoredBuckets, IntelligenceProfiles, IntelligenceModels, IntelligenceTrainingRuns, IntelligencePipelineActive, IntelligencePipelineFailed, DatabaseRows, HTTPRouteDuration, MLTrainingExamples, MLTrainingPositiveExamples, MLTrainingNegativeExamples, MLAnomalyScoreAverage, MLPredictionConfidenceAverage, IntelligenceDriftScore, IntelligenceCalibrationError, IntelligenceHealth)
}
