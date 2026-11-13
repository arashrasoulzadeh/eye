package service

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// DashboardCacheService maintains a cached snapshot of dashboard metrics.
// It's updated by a background job on a configurable interval.
type DashboardCacheService struct {
	DB *gorm.DB
}

// Refresh recomputes the dashboard metrics and stores them in the cache.
func (s *DashboardCacheService) Refresh() error {
	var events int64
	s.DB.Model(&model.Event{}).Where("is_aggregated = ?", false).Count(&events)

	var sessions int64
	s.DB.Model(&model.Session{}).Count(&sessions)
	now := time.Now()
	var onlineUsers int64
	s.DB.Model(&model.Session{}).Where("last_seen >= ?", now.Add(-5*time.Minute)).Count(&onlineUsers)
	var cohortSessions []model.Session
	s.DB.Where("started_at < ? AND started_at >= ?", now.Add(-14*24*time.Hour), now.Add(-21*24*time.Hour)).Find(&cohortSessions)
	retentionRate := 0.0
	if len(cohortSessions) > 0 {
		ids := make([]string, 0, len(cohortSessions))
		for _, session := range cohortSessions {
			ids = append(ids, session.ID)
		}
		var retained int64
		s.DB.Model(&model.Event{}).Where("session_id IN ? AND client_ts >= ?", ids, now.Add(-7*24*time.Hour).UnixMilli()).Distinct("session_id").Count(&retained)
		retentionRate = float64(retained) / float64(len(cohortSessions))
	}

	var tenants int64
	s.DB.Model(&model.Tenant{}).Count(&tenants)

	// Get events for hourly breakdown (last 1000 events)
	var recentEvents []model.Event
	s.DB.Where("is_aggregated = ?", false).Order("client_ts DESC").Limit(1000).Find(&recentEvents)

	hourly := make(map[string]int64)
	for _, e := range recentEvents {
		d := time.UnixMilli(e.ClientTS)
		label := d.Format("15:00")
		hourly[label]++
	}

	type ChartPoint struct {
		Label string `json:"label"`
		Value int64  `json:"value"`
	}

	var hourlyPoints []ChartPoint
	for h := 0; h < 24; h++ {
		label := time.Date(0, 1, 1, h, 0, 0, 0, time.UTC).Format("15:00")
		hourlyPoints = append(hourlyPoints, ChartPoint{Label: label, Value: hourly[label]})
	}

	// Get tenants for tenant breakdown
	var tenantList []model.Tenant
	s.DB.Find(&tenantList)
	tenantNames := make(map[string]string)
	for _, t := range tenantList {
		tenantNames[t.ID] = t.Name
	}

	tenantCounts := make(map[string]int64)
	for _, e := range recentEvents {
		tenantCounts[e.TenantID]++
	}

	var tenantPoints []ChartPoint
	for id, count := range tenantCounts {
		name := tenantNames[id]
		if name == "" {
			name = id[:8]
		}
		tenantPoints = append(tenantPoints, ChartPoint{Label: name, Value: count})
	}

	// Sort by value descending and take top 8
	for i := 0; i < len(tenantPoints)-1; i++ {
		for j := i + 1; j < len(tenantPoints); j++ {
			if tenantPoints[i].Value < tenantPoints[j].Value {
				tenantPoints[i], tenantPoints[j] = tenantPoints[j], tenantPoints[i]
			}
		}
	}
	if len(tenantPoints) > 8 {
		tenantPoints = tenantPoints[:8]
	}

	hourlyJSON, _ := json.Marshal(hourlyPoints)
	tenantJSON, _ := json.Marshal(tenantPoints)

	cache := &model.DashboardCache{
		ID:             1,
		Events:         events,
		Sessions:       sessions,
		Tenants:        tenants,
		Users:          sessions,
		OnlineUsers:    onlineUsers,
		RetentionRate:  retentionRate,
		EventsByHour:   string(hourlyJSON),
		EventsByTenant: string(tenantJSON),
		UpdatedAt:      time.Now().UnixMilli(),
	}

	return s.DB.Save(cache).Error
}

// Get returns the cached dashboard metrics.
func (s *DashboardCacheService) Get() (*model.DashboardCache, error) {
	var cache model.DashboardCache
	if err := s.DB.First(&cache, 1).Error; err != nil {
		return nil, err
	}
	return &cache, nil
}

// GetRange overlays the cached dashboard with metrics calculated for a caller's
// selected time window. The background cache remains the fast fallback for the
// unfiltered dashboard, while range-aware views always reflect their query.
func (s *DashboardCacheService) GetRange(from, to int64) (*model.DashboardCache, error) {
	cache, err := s.Get()
	if err != nil {
		return nil, err
	}
	var events int64
	if err := s.DB.Model(&model.Event{}).Where("is_aggregated = ? AND client_ts >= ? AND client_ts <= ?", false, from, to).Count(&events).Error; err != nil {
		return nil, err
	}
	var sessions int64
	if err := s.DB.Model(&model.Event{}).Where("is_aggregated = ? AND client_ts >= ? AND client_ts <= ?", false, from, to).Distinct("session_id").Count(&sessions).Error; err != nil {
		return nil, err
	}
	var rows []model.Event
	// Load the complete selected window. Truncating this to the newest 1,000
	// events makes the chart appear to ignore the global time range whenever a
	// busy tenant has more than 1,000 events in it.
	if err := s.DB.Where("is_aggregated = ? AND client_ts >= ? AND client_ts <= ?", false, from, to).Order("client_ts DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	cache.Events = events
	cache.Sessions = sessions
	cache.Users = sessions
	cache.EventsByTenant = dashboardTenantBreakdown(s.DB, rows)
	cache.EventsByHour = dashboardHourBreakdown(rows, from, to)
	cache.UpdatedAt = time.Now().UnixMilli()
	return cache, nil
}

func dashboardHourBreakdown(events []model.Event, from, to int64) string {
	counts := make(map[int64]int64)
	for _, event := range events {
		counts[time.UnixMilli(event.ClientTS).Truncate(time.Hour).UnixMilli()]++
	}
	type point struct {
		Label string `json:"label"`
		Value int64  `json:"value"`
	}
	if from >= to {
		encoded, _ := json.Marshal([]point{})
		return string(encoded)
	}
	start := time.UnixMilli(from).Truncate(time.Hour)
	end := time.UnixMilli(to).Truncate(time.Hour)
	// A rolling 24-hour range can cross midnight. Include the date at exactly
	// 24 hours too, otherwise labels such as 00:00 repeat and make the chart
	// look like a calendar-day series rather than the selected global window.
	includeDate := to-from >= int64(24*time.Hour/time.Millisecond)
	points := make([]point, 0, int(end.Sub(start)/time.Hour)+1)
	for bucket := start; !bucket.After(end); bucket = bucket.Add(time.Hour) {
		label := bucket.Format("15:00")
		if includeDate {
			label = bucket.Format("Jan 2 15:00")
		}
		points = append(points, point{label, counts[bucket.UnixMilli()]})
	}
	encoded, _ := json.Marshal(points)
	return string(encoded)
}

func dashboardTenantBreakdown(db *gorm.DB, events []model.Event) string {
	var tenants []model.Tenant
	db.Find(&tenants)
	names := make(map[string]string)
	for _, tenant := range tenants {
		names[tenant.ID] = tenant.Name
	}
	counts := make(map[string]int64)
	for _, event := range events {
		counts[event.TenantID]++
	}
	type point struct {
		Label string `json:"label"`
		Value int64  `json:"value"`
	}
	points := make([]point, 0, len(counts))
	for id, count := range counts {
		label := names[id]
		if label == "" {
			label = id
			if len(label) > 8 {
				label = label[:8]
			}
		}
		points = append(points, point{label, count})
	}
	for i := 0; i < len(points)-1; i++ {
		for j := i + 1; j < len(points); j++ {
			if points[i].Value < points[j].Value {
				points[i], points[j] = points[j], points[i]
			}
		}
	}
	if len(points) > 8 {
		points = points[:8]
	}
	encoded, _ := json.Marshal(points)
	return string(encoded)
}
