package test

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func TestDetectZScoreStableHistoryNoAnomaly(t *testing.T) {
	history := []int64{10, 11, 9, 10, 10}
	anomalous, _, _, _ := service.DetectZScore(history, 10, 3.0)
	if anomalous {
		t.Fatal("a value in line with stable history should not be flagged")
	}
}

func TestDetectZScoreSpikeIsFlagged(t *testing.T) {
	history := []int64{10, 11, 9, 10, 10}
	// mean ~10, stddev ~0.63 over this history; 1000 is enormously far above.
	anomalous, mean, stddev, z := service.DetectZScore(history, 1000, 3.0)
	if !anomalous {
		t.Fatalf("expected a huge spike to be flagged (mean=%v stddev=%v z=%v)", mean, stddev, z)
	}
	if z <= 3.0 {
		t.Fatalf("expected z-score above threshold, got %v", z)
	}
}

func TestDetectZScoreInsufficientHistoryNeverFlags(t *testing.T) {
	anomalous, _, _, _ := service.DetectZScore([]int64{10}, 1000, 3.0)
	if anomalous {
		t.Fatal("a single history point is insufficient to compute a meaningful stddev; must not flag")
	}
	anomalous, _, _, _ = service.DetectZScore(nil, 1000, 3.0)
	if anomalous {
		t.Fatal("empty history must not flag")
	}
}

func TestDetectZScoreFlatHistoryDifferentLatestIsFlagged(t *testing.T) {
	history := []int64{10, 10, 10, 10}
	anomalous, _, stddev, _ := service.DetectZScore(history, 50, 3.0)
	if stddev != 0 {
		t.Fatalf("expected zero stddev for perfectly flat history, got %v", stddev)
	}
	if !anomalous {
		t.Fatal("any deviation from a perfectly flat history should be flagged")
	}
}

func TestDetectZScoreFlatHistorySameLatestNotFlagged(t *testing.T) {
	history := []int64{10, 10, 10, 10}
	anomalous, _, _, _ := service.DetectZScore(history, 10, 3.0)
	if anomalous {
		t.Fatal("matching a perfectly flat history exactly should not be flagged")
	}
}

func newAnomalyTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.AggregateCount{}, &service.MetricSample{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAnomalyServiceSampleTakesDeltaFromCumulative(t *testing.T) {
	db := newAnomalyTestDB(t)
	svc := service.NewAnomalyService(db)
	base := time.Unix(1000, 0)

	db.Create(&model.AggregateCount{TenantID: "t1", EventName: "click", DimsKey: "", Count: 10})
	if err := svc.Sample(base); err != nil {
		t.Fatalf("first sample: %v", err)
	}
	var s1 service.MetricSample
	db.Where("tenant_id = ? AND event_name = ? AND bucket = ?", "t1", "click", base.Unix()).First(&s1)
	if s1.Count != 10 {
		t.Fatalf("first sample's delta should equal the full cumulative (no prior sample), got %v", s1.Count)
	}

	db.Model(&model.AggregateCount{}).Where("tenant_id = ?", "t1").Update("count", 25)
	next := base.Add(5 * time.Minute)
	if err := svc.Sample(next); err != nil {
		t.Fatalf("second sample: %v", err)
	}
	var s2 service.MetricSample
	db.Where("tenant_id = ? AND event_name = ? AND bucket = ?", "t1", "click", next.Unix()).First(&s2)
	if s2.Count != 15 {
		t.Fatalf("second sample's delta should be 25-10=15, got %v", s2.Count)
	}
}

func TestAnomalyServiceSampleIsIdempotentWithinSameBucket(t *testing.T) {
	db := newAnomalyTestDB(t)
	svc := service.NewAnomalyService(db)
	bucket := time.Unix(2000, 0)

	db.Create(&model.AggregateCount{TenantID: "t1", EventName: "click", DimsKey: "", Count: 5})
	if err := svc.Sample(bucket); err != nil {
		t.Fatalf("sample: %v", err)
	}
	if err := svc.Sample(bucket); err != nil {
		t.Fatalf("re-sample same bucket: %v", err)
	}
	var count int64
	db.Model(&service.MetricSample{}).Where("tenant_id = ? AND event_name = ? AND bucket = ?", "t1", "click", bucket.Unix()).Count(&count)
	if count != 1 {
		t.Fatalf("re-sampling the same bucket must overwrite, not duplicate; got %d rows", count)
	}
}

func TestAnomalyServiceCurrentAnomaliesFlagsASpike(t *testing.T) {
	db := newAnomalyTestDB(t)
	svc := service.NewAnomalyService(db)
	svc.WindowSize = 3
	svc.ZThreshold = 3.0

	// Five stable buckets (count=10 each via cumulative +10 per tick) then a
	// spike on the sixth, through the real Sample() + AggregateCount path.
	db.Create(&model.AggregateCount{TenantID: "t1", EventName: "click", DimsKey: "", Count: 0})
	cumulative := int64(0)
	deltas := []int64{10, 10, 10, 10, 10, 500}
	for i, d := range deltas {
		cumulative += d
		db.Model(&model.AggregateCount{}).Where("tenant_id = ?", "t1").Update("count", cumulative)
		if err := svc.Sample(time.Unix(int64(i)*300, 0)); err != nil {
			t.Fatalf("sample %d: %v", i, err)
		}
	}

	anomalies, err := svc.CurrentAnomalies()
	if err != nil {
		t.Fatalf("CurrentAnomalies: %v", err)
	}
	if len(anomalies) != 1 {
		t.Fatalf("expected exactly one anomaly, got %d: %+v", len(anomalies), anomalies)
	}
	if anomalies[0].TenantID != "t1" || anomalies[0].EventName != "click" || anomalies[0].Count != 500 {
		t.Fatalf("unexpected anomaly: %+v", anomalies[0])
	}
}

func TestAnomalyServiceCurrentAnomaliesNoFlagWhenStable(t *testing.T) {
	db := newAnomalyTestDB(t)
	svc := service.NewAnomalyService(db)
	svc.WindowSize = 3

	db.Create(&model.AggregateCount{TenantID: "t1", EventName: "click", DimsKey: "", Count: 0})
	cumulative := int64(0)
	for i := 0; i < 6; i++ {
		cumulative += 10
		db.Model(&model.AggregateCount{}).Where("tenant_id = ?", "t1").Update("count", cumulative)
		if err := svc.Sample(time.Unix(int64(i)*300, 0)); err != nil {
			t.Fatalf("sample %d: %v", i, err)
		}
	}

	anomalies, err := svc.CurrentAnomalies()
	if err != nil {
		t.Fatalf("CurrentAnomalies: %v", err)
	}
	if len(anomalies) != 0 {
		t.Fatalf("expected no anomalies for stable load, got %+v", anomalies)
	}
}

func TestAnomalyServiceCurrentAnomaliesInsufficientHistoryNoFalsePositive(t *testing.T) {
	db := newAnomalyTestDB(t)
	svc := service.NewAnomalyService(db)
	svc.WindowSize = 5

	db.Create(&model.AggregateCount{TenantID: "t1", EventName: "click", DimsKey: "", Count: 0})
	cumulative := int64(0)
	// Only 2 samples total — fewer than WindowSize+1 — must never flag.
	for i := 0; i < 2; i++ {
		cumulative += 1000
		db.Model(&model.AggregateCount{}).Where("tenant_id = ?", "t1").Update("count", cumulative)
		if err := svc.Sample(time.Unix(int64(i)*300, 0)); err != nil {
			t.Fatalf("sample %d: %v", i, err)
		}
	}

	anomalies, err := svc.CurrentAnomalies()
	if err != nil {
		t.Fatalf("CurrentAnomalies: %v", err)
	}
	if len(anomalies) != 0 {
		t.Fatalf("expected no anomalies with insufficient history, got %+v", anomalies)
	}
}
