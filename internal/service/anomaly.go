package service

import (
	"context"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// MetricSample is one periodic, per-bucket snapshot of how many events a
// (tenant, event_name) pair received — the delta of the Phase 7
// AggregateCount running total since the previous sample. AggregateCount
// itself is a lifetime total (see its doc comment), so a rolling window of
// recent *deltas* needs its own small table; this is that table.
type MetricSample struct {
	TenantID   string `gorm:"primaryKey"`
	EventName  string `gorm:"primaryKey"`
	Bucket     int64  `gorm:"primaryKey"` // unix seconds, truncated to the sampling interval
	Count      int64  // delta since the previous sample, i.e. this bucket's count
	Cumulative int64  // the AggregateCount total this sample was taken against
}

// AnomalyService periodically samples AggregateCount into MetricSample
// buckets and flags the latest bucket as anomalous via z-score against the
// rolling mean/stddev of the buckets before it. This is deliberately a
// basic, named statistical method (z-score over a rolling window), not any
// kind of learned/ML model.
type AnomalyService struct {
	DB *gorm.DB
	// WindowSize is how many prior buckets are used to compute the rolling
	// mean/stddev that the latest bucket is compared against.
	WindowSize int
	// ZThreshold is how many standard deviations above the rolling mean a
	// bucket must be to be flagged.
	ZThreshold float64
}

func NewAnomalyService(db *gorm.DB) *AnomalyService {
	return &AnomalyService{DB: db, WindowSize: 5, ZThreshold: 3.0}
}

// Sample takes one snapshot for every distinct (tenant_id, event_name) that
// has any AggregateCount rows, recording the delta since each pair's most
// recent prior sample (0 the first time a pair is seen). bucket should be a
// stable, truncated timestamp (the caller's sampling tick), so repeated
// calls within the same tick are idempotent overwrites rather than
// duplicate buckets.
func (a *AnomalyService) Sample(bucket time.Time) error {
	type row struct {
		TenantID  string
		EventName string
		Total     int64
	}
	var rows []row
	if err := a.DB.Model(&model.AggregateCount{}).
		Select("tenant_id, event_name, COALESCE(SUM(count), 0) as total").
		Group("tenant_id, event_name").
		Scan(&rows).Error; err != nil {
		return err
	}
	bucketSec := bucket.Unix()
	for _, r := range rows {
		var prev MetricSample
		err := a.DB.Where("tenant_id = ? AND event_name = ?", r.TenantID, r.EventName).
			Order("bucket DESC").First(&prev).Error
		prevCumulative := int64(0)
		if err == nil {
			prevCumulative = prev.Cumulative
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
		delta := r.Total - prevCumulative
		if delta < 0 {
			delta = 0
		}
		// Explicit upsert rather than Save(): a composite primary key with
		// any zero-value field (not the case here, since bucketSec/TenantID/
		// EventName are all non-zero in practice, but kept consistent with
		// UpsertAggregateCount's established pattern to avoid the same class
		// of GORM Save()-treats-zero-PK-as-new-record bug) makes Where+
		// Create/Updates the safer default for any table keyed this way.
		var existing MetricSample
		err = a.DB.Where("tenant_id = ? AND event_name = ? AND bucket = ?", r.TenantID, r.EventName, bucketSec).
			First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			sample := MetricSample{TenantID: r.TenantID, EventName: r.EventName, Bucket: bucketSec, Count: delta, Cumulative: r.Total}
			if err := a.DB.Create(&sample).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err := a.DB.Model(&MetricSample{}).
			Where("tenant_id = ? AND event_name = ? AND bucket = ?", r.TenantID, r.EventName, bucketSec).
			Updates(map[string]any{"count": delta, "cumulative": r.Total}).Error; err != nil {
			return err
		}
	}
	return nil
}

// Anomaly describes one (tenant, event_name) pair whose latest sampled
// bucket is statistically unusual relative to its own recent history.
type Anomaly struct {
	TenantID  string  `json:"tenant_id"`
	EventName string  `json:"event_name"`
	Count     int64   `json:"count"`
	Mean      float64 `json:"mean"`
	StdDev    float64 `json:"stddev"`
	ZScore    float64 `json:"z_score"`
}

// DetectZScore is the pure, DB-free statistical core: given a history of
// prior bucket counts (oldest first) and the latest bucket's count, it
// reports whether latest is more than threshold standard deviations above
// the mean of history. Needs at least 2 history points to compute a
// meaningful stddev; with fewer, it never flags (insufficient history, not a
// false positive).
func DetectZScore(history []int64, latest int64, threshold float64) (anomalous bool, mean, stddev, z float64) {
	if len(history) < 2 {
		return false, 0, 0, 0
	}
	var sum float64
	for _, h := range history {
		sum += float64(h)
	}
	mean = sum / float64(len(history))
	var variance float64
	for _, h := range history {
		d := float64(h) - mean
		variance += d * d
	}
	variance /= float64(len(history))
	stddev = math.Sqrt(variance)
	if stddev == 0 {
		// A latest value different from an otherwise perfectly flat history
		// is an enormous number of stddevs away; report it as such without
		// an actual +Inf (not JSON-representable — encoding/json errors out
		// on it, which would otherwise silently empty the /anomalies
		// response body) by substituting a tiny epsilon for the zero stddev.
		if float64(latest) != mean {
			const epsilon = 1e-9
			return true, mean, 0, (float64(latest) - mean) / epsilon
		}
		return false, mean, 0, 0
	}
	z = (float64(latest) - mean) / stddev
	return z > threshold, mean, stddev, z
}

// CurrentAnomalies evaluates every (tenant, event_name) pair with enough
// sampled history and returns the ones whose latest bucket is currently
// flagged.
func (a *AnomalyService) CurrentAnomalies() ([]Anomaly, error) {
	type pair struct {
		TenantID  string
		EventName string
	}
	var pairs []pair
	if err := a.DB.Model(&MetricSample{}).Distinct("tenant_id", "event_name").Scan(&pairs).Error; err != nil {
		return nil, err
	}
	var out []Anomaly
	for _, p := range pairs {
		var samples []MetricSample
		if err := a.DB.Where("tenant_id = ? AND event_name = ?", p.TenantID, p.EventName).
			Order("bucket ASC").Find(&samples).Error; err != nil {
			return nil, err
		}
		if len(samples) < a.WindowSize+1 {
			continue
		}
		latest := samples[len(samples)-1]
		historyStart := len(samples) - 1 - a.WindowSize
		history := make([]int64, 0, a.WindowSize)
		for _, s := range samples[historyStart : len(samples)-1] {
			history = append(history, s.Count)
		}
		flagged, mean, stddev, z := DetectZScore(history, latest.Count, a.ZThreshold)
		if flagged {
			out = append(out, Anomaly{TenantID: p.TenantID, EventName: p.EventName, Count: latest.Count, Mean: mean, StdDev: stddev, ZScore: z})
		}
	}
	return out, nil
}

// CurrentAnomaliesPage returns a bounded, searchable view for admin tables.
// Detection remains centralized in CurrentAnomalies so the page cannot drift
// from the alerting path's anomaly semantics.
func (a *AnomalyService) CurrentAnomaliesPage(filter, direction string, limit, offset int) ([]Anomaly, int, error) {
	all, err := a.CurrentAnomalies()
	if err != nil {
		return nil, 0, err
	}
	needle := strings.ToLower(strings.TrimSpace(filter))
	filtered := all[:0]
	for _, item := range all {
		if needle == "" || strings.Contains(strings.ToLower(item.TenantID), needle) || strings.Contains(strings.ToLower(item.EventName), needle) {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ZScore > filtered[j].ZScore })
	if direction == "asc" {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].ZScore < filtered[j].ZScore })
	}
	total := len(filtered)
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []Anomaly{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

// RunAnomalySamplingJob periodically snapshots AggregateCount into
// MetricSample buckets so CurrentAnomalies has rolling history to compare
// against. Like the other Phase 6/7/8 ticker jobs, this is a polling
// snapshot, not a real-time streaming computation.
func RunAnomalySamplingJob(ctx context.Context, svc *AnomalyService, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.Sample(time.Now().Truncate(interval)); err != nil && logger != nil {
				logger.Error("anomaly sampling job failed", "err", err)
			}
		}
	}
}
