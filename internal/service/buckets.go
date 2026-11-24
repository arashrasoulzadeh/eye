package service

import (
	"context"
	"encoding/json"
	"golang.org/x/sync/singleflight"
	"sort"
	"sync"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BucketPoint is a pre-aggregated event count for one time bucket.
type BucketPoint struct {
	Bucket int64 `json:"bucket"`
	Count  int   `json:"count"`
}

type bucketCacheEntry struct {
	expires time.Time
	points  []BucketPoint
}

// BucketService provides shared, short-lived time-bucketed event counts. It is
// intentionally backed by raw events for correctness; the cache prevents all
// analytics surfaces from repeating the same scan during refreshes.
type BucketService struct {
	DB    *gorm.DB
	mu    sync.Mutex
	cache map[string]bucketCacheEntry
	flights singleflight.Group
	once sync.Once
	gate chan struct{}
	queue chan struct{}
}

func bucketWidth(name string) (int64, bool) {
	switch name {
	case "minute":
		return 60_000, true
	case "10minute":
		return 10 * 60_000, true
	case "hour":
		return 60 * 60_000, true
	default:
		return 0, false
	}
}

func (s *BucketService) Events(tenantID, eventName, bucket string, fromTS, toTS int64) ([]BucketPoint, error) {
	return s.EventsContext(context.Background(), tenantID, eventName, bucket, fromTS, toTS)
}

func (s *BucketService) EventsContext(ctx context.Context, tenantID, eventName, bucket string, fromTS, toTS int64) ([]BucketPoint, error) {
	keyBytes, _ := json.Marshal(struct { Tenant, Event, Bucket string; From, To int64 }{tenantID, eventName, bucket, fromTS, toTS})
	key := string(keyBytes)
	s.once.Do(func(){ s.gate = make(chan struct{}, 4); s.queue = make(chan struct{}, 64) })
	select { case s.queue <- struct{}{}: defer func(){ <-s.queue }(); default: return nil, gorm.ErrInvalidData }
	ch := s.flights.DoChan(key, func() (any, error) {
		work, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		select { case s.gate <- struct{}{}: defer func(){ <-s.gate }(); case <-work.Done(): return nil, work.Err() }
		return s.events(work, key, tenantID, eventName, bucket, fromTS, toTS)
	})
	select {
	case <-ctx.Done(): return nil, ctx.Err()
	case result := <-ch:
		if result.Err != nil { return nil, result.Err }
		points := result.Val.([]BucketPoint)
		return append(make([]BucketPoint, 0, len(points)), points...), nil
	}
}

func (s *BucketService) events(ctx context.Context, key, tenantID, eventName, bucket string, fromTS, toTS int64) ([]BucketPoint, error) {
	width, ok := bucketWidth(bucket)
	if !ok {
		return nil, gorm.ErrInvalidData
	}
	s.mu.Lock()
	if entry, found := s.cache[key]; found && time.Now().Before(entry.expires) {
		points := append([]BucketPoint(nil), entry.points...)
		s.mu.Unlock()
		return points, nil
	}
	s.mu.Unlock()

	firstFull := fromTS - fromTS%width
	if firstFull < fromTS {
		firstFull += width
	}
	lastFullEnd := toTS - toTS%width
	query := s.DB.WithContext(ctx).Where("bucket = ? AND bucket_start >= ? AND bucket_start < ?", bucket, firstFull, lastFullEnd)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if eventName != "" {
		query = query.Where("event_name = ?", eventName)
	}
	var rows []model.EventBucket
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[int64]int)
	for _, row := range rows {
		counts[row.BucketStart] += int(row.Count)
	}
	// Durable buckets contain the entire interval. Count partial edges from
	// raw events so they respect the exact requested bounds.
	edge := s.DB.WithContext(ctx).Model(&model.Event{}).Where("is_aggregated = ? AND client_ts >= ? AND client_ts <= ?", false, fromTS, toTS).
		Where("(client_ts < ? OR client_ts >= ?)", firstFull, lastFullEnd)
	if tenantID != "" {
		edge = edge.Where("tenant_id = ?", tenantID)
	}
	if eventName != "" {
		edge = edge.Where("event_name = ?", eventName)
	}
	var edges []struct {
		Bucket int64
		Count  int
	}
	if err := edge.Select("client_ts - (client_ts % ?) AS bucket, COUNT(*) AS count", width).
		Group("bucket").Scan(&edges).Error; err != nil {
		return nil, err
	}
	for _, row := range edges {
		counts[row.Bucket] += row.Count
	}
	points := make([]BucketPoint, 0, len(counts))
	for bucketStart, count := range counts {
		points = append(points, BucketPoint{Bucket: bucketStart, Count: count})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Bucket < points[j].Bucket })
	s.mu.Lock()
	if s.cache == nil {
		s.cache = make(map[string]bucketCacheEntry)
	}
	if len(s.cache) >= 512 {
		for oldKey := range s.cache {
			delete(s.cache, oldKey)
			break
		}
	}
	s.cache[key] = bucketCacheEntry{expires: time.Now().Add(10 * time.Second), points: append([]BucketPoint(nil), points...)}
	s.mu.Unlock()
	return points, nil
}

// UpsertEventBuckets updates all durable resolutions for one accepted event.
func UpsertEventBuckets(db *gorm.DB, events ...*model.Event) error {
	type bucketKey struct {
		tenant, event, name string
		start               int64
	}
	counts := make(map[bucketKey]int64)
	for _, event := range events {
		if event.IsAggregated {
			continue
		}
		for _, spec := range []struct {
			name  string
			width int64
		}{{"minute", 60_000}, {"10minute", 600_000}, {"hour", 3_600_000}} {
			start := (event.ClientTS / spec.width) * spec.width
			counts[bucketKey{event.TenantID, event.EventName, spec.name, start}]++
		}
	}
	if len(counts) == 0 {
		return nil
	}
	now := time.Now()
	rows := make([]model.EventBucket, 0, len(counts))
	for key, count := range counts {
		rows = append(rows, model.EventBucket{TenantID: key.tenant, EventName: key.event, Bucket: key.name, BucketStart: key.start, Count: count, UpdatedAt: now})
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.TenantID != b.TenantID {
			return a.TenantID < b.TenantID
		}
		if a.EventName != b.EventName {
			return a.EventName < b.EventName
		}
		if a.Bucket != b.Bucket {
			return a.Bucket < b.Bucket
		}
		return a.BucketStart < b.BucketStart
	})
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "event_name"}, {Name: "bucket"}, {Name: "bucket_start"}},
		DoUpdates: clause.Assignments(map[string]any{
			"count":      gorm.Expr("event_buckets.count + excluded.count"),
			"updated_at": gorm.Expr("excluded.updated_at"),
		}),
	}).Create(&rows).Error
}

// Backfill creates missing durable buckets for events ingested before the
// bucket table existed. Existing rows are rebuilt idempotently at startup.
func (s *BucketService) Backfill() error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.EventBucket{}).Error; err != nil {
			return err
		}
		for _, spec := range []struct { name string; width int64 }{
			{"minute", 60_000}, {"10minute", 600_000}, {"hour", 3_600_000},
		} {
			if err := tx.Exec("INSERT INTO event_buckets (tenant_id, event_name, bucket, bucket_start, count, updated_at) SELECT tenant_id, event_name, ?, client_ts - (client_ts % ?), COUNT(*), ? FROM events WHERE is_aggregated = ? GROUP BY tenant_id, event_name, 4", spec.name, spec.width, time.Now(), false).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
