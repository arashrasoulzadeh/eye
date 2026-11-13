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
	width, ok := bucketWidth(bucket)
	if !ok {
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

	query := s.DB.Model(&model.Event{}).Select("client_ts").Where("tenant_id = ? AND client_ts >= ? AND client_ts <= ?", tenantID, fromTS, toTS)
	if eventName != "" {
		query = query.Where("event_name = ?", eventName)
	}
	var rows []struct{ ClientTS int64 }
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[int64]int)
	for _, row := range rows {
		counts[(row.ClientTS/width)*width]++
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
