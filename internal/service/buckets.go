package service

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
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
	if _, ok := bucketWidth(bucket); !ok {
		return nil, gorm.ErrInvalidData
	}
	key := fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%d", tenantID, eventName, bucket, fromTS, toTS)
	s.mu.Lock()
	if entry, found := s.cache[key]; found && time.Now().Before(entry.expires) {
		points := append([]BucketPoint(nil), entry.points...)
		s.mu.Unlock()
		return points, nil
	}
	s.mu.Unlock()

	var rows []model.EventBucket
	query := s.DB.Where("tenant_id = ? AND bucket = ? AND bucket_start >= ? AND bucket_start <= ?", tenantID, bucket, fromTS, toTS)
	if eventName != "" {
		query = query.Where("event_name = ?", eventName)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[int64]int)
	for _, row := range rows {
		counts[row.BucketStart] += int(row.Count)
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
func UpsertEventBuckets(db *gorm.DB, event *model.Event) error {
	for _, spec := range []struct {
		name  string
		width int64
	}{{"minute", 60_000}, {"10minute", 600_000}, {"hour", 3_600_000}} {
		start := (event.ClientTS / spec.width) * spec.width
		var row model.EventBucket
		err := db.Where("tenant_id = ? AND event_name = ? AND bucket = ? AND bucket_start = ?", event.TenantID, event.EventName, spec.name, start).First(&row).Error
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&model.EventBucket{TenantID: event.TenantID, EventName: event.EventName, Bucket: spec.name, BucketStart: start, Count: 1, UpdatedAt: time.Now()}).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err := db.Model(&row).Updates(map[string]any{"count": row.Count + 1, "updated_at": time.Now()}).Error; err != nil {
			return err
		}
	}
	return nil
}

// Backfill creates missing durable buckets for events ingested before the
// bucket table existed. Existing rows are rebuilt idempotently at startup.
func (s *BucketService) Backfill() error {
	var events []model.Event
	if err := s.DB.Select("tenant_id, event_name, client_ts").Find(&events).Error; err != nil {
		return err
	}
	for i := range events {
		if err := UpsertEventBuckets(s.DB, &events[i]); err != nil {
			return err
		}
	}
	return nil
}
