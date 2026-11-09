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
	s.DB.Model(&model.Event{}).Count(&events)

	var sessions int64
	s.DB.Model(&model.Session{}).Count(&sessions)

	var tenants int64
	s.DB.Model(&model.Tenant{}).Count(&tenants)

	// Get events for hourly breakdown (last 1000 events)
	var recentEvents []model.Event
	s.DB.Order("client_ts DESC").Limit(1000).Find(&recentEvents)

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