package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
)

type QuestionService struct{ DB *gorm.DB }

func questionID() string { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }

func (s *QuestionService) List(tenantID string) ([]model.Question, error) {
	var out []model.Question
	err := s.DB.Where("tenant_id = ?", tenantID).Order("updated_at DESC").Find(&out).Error
	return out, err
}

func (s *QuestionService) ListPage(tenantID, filter, direction string, limit, offset int) ([]model.Question, int64, error) {
	var out []model.Question
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	countQuery := s.DB.Model(&model.Question{}).Where("tenant_id = ?", tenantID)
	query := s.DB.Where("tenant_id = ?", tenantID)
	if value := strings.TrimSpace(filter); value != "" {
		like := "%" + value + "%"
		countQuery = countQuery.Where("name LIKE ? OR event_names LIKE ?", like, like)
		query = query.Where("name LIKE ? OR event_names LIKE ?", like, like)
	}
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "updated_at desc"
	if strings.EqualFold(direction, "asc") {
		order = "updated_at asc"
	}
	if err := query.Order(order).Limit(limit).Offset(offset).Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *QuestionService) Save(q *model.Question) (*model.Question, error) {
	if q.TenantID == "" || q.Name == "" || (q.EventName == "" && q.EventNames == "") {
		return nil, errors.New("tenant_id, name and event_name are required")
	}
	if q.EventNames == "" {
		q.EventNames = q.EventName
	}
	if q.EventName == "" {
		q.EventName = strings.Split(q.EventNames, ",")[0]
	}
	if q.MatchMode == "" {
		q.MatchMode = "any"
	}
	if q.MatchMode != "any" && q.MatchMode != "all" {
		return nil, errors.New("match_mode must be any or all")
	}
	// Chart and time are selected by the Graphs view. Keep legacy columns
	// populated for old clients and database rows, but do not require them.
	if q.ChartType == "" {
		q.ChartType = "line"
	}
	if q.RangeMode == "" {
		q.RangeMode = "relative"
	}
	if q.RelativeDays < 1 {
		q.RelativeDays = 1
	}
	if q.RangeMode != "relative" && q.RangeMode != "absolute" {
		return nil, errors.New("range_mode must be relative or absolute")
	}
	if q.ChartType != "line" && q.ChartType != "bar" {
		return nil, errors.New("chart_type must be line or bar")
	}
	if q.RangeMode == "relative" && q.RelativeDays < 1 {
		return nil, errors.New("relative_days must be positive")
	}
	if q.RangeMode == "absolute" && (q.FromTS <= 0 || q.ToTS <= q.FromTS) {
		return nil, errors.New("absolute range is invalid")
	}
	now := time.Now().Unix()
	q.UpdatedAt = now
	if q.ID == "" {
		q.ID = questionID()
		q.CreatedAt = now
		if err := s.DB.Create(q).Error; err != nil {
			return nil, err
		}
	} else if err := s.DB.Model(&model.Question{}).Where("id = ? AND tenant_id = ?", q.ID, q.TenantID).Updates(q).Error; err != nil {
		return nil, err
	}
	return q, nil
}

func (s *QuestionService) Delete(tenantID, id string) error {
	return s.DB.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&model.Question{}).Error
}

type GraphResult struct {
	Question model.Question `json:"question"`
	FromTS   int64          `json:"from_ts"`
	ToTS     int64          `json:"to_ts"`
	Points   []ChartPoint   `json:"points"`
}
type ChartPoint struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

func (s *QuestionService) Results(tenantID, id string, now time.Time, overrideFrom, overrideTo int64) (*GraphResult, error) {
	var q model.Question
	if err := s.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&q).Error; err != nil {
		return nil, err
	}
	to := now.UnixMilli()
	from := q.FromTS
	if q.RangeMode == "relative" {
		to = now.UnixMilli()
		from = now.Add(-time.Duration(q.RelativeDays) * 24 * time.Hour).UnixMilli()
	}
	if overrideFrom > 0 && overrideTo > overrideFrom {
		from, to = overrideFrom, overrideTo
	}
	names := strings.Split(q.EventNames, ",")
	allowed := map[string]bool{}
	for _, name := range names {
		allowed[strings.TrimSpace(name)] = true
	}
	var events []model.Event
	if err := s.DB.Where("tenant_id = ? AND client_ts >= ? AND client_ts <= ?", tenantID, from, to).Find(&events).Error; err != nil {
		return nil, err
	}
	buckets := map[string]int{}
	for _, e := range events {
		if !allowed[e.EventName] {
			continue
		}
		d := time.UnixMilli(e.ClientTS).UTC()
		key := d.Format("2006-01-02")
		buckets[key]++
	}
	points := make([]ChartPoint, 0, len(buckets))
	for k, v := range buckets {
		points = append(points, ChartPoint{Label: k, Value: v})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Label < points[j].Label })
	return &GraphResult{Question: q, FromTS: from, ToTS: to, Points: points}, nil
}
