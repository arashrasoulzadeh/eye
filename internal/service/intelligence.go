package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
)

type UserIntelligenceService struct{ DB *gorm.DB }

var allowedIntelligenceLabels = map[string]bool{"buyer": true, "potential_buyer": true, "returning_user": true, "new_user": true, "bot": true, "automation": true, "ai_crawler": true, "high_intent": true, "inactive": true}

func (s *UserIntelligenceService) AddLabel(tenantID, fingerprint, label, source string) error {
	if !allowedIntelligenceLabels[label] {
		return fmt.Errorf("unsupported intelligence label: %s", label)
	}
	if source == "" {
		source = "admin"
	}
	id := tenantID + ":" + fingerprint + ":" + label
	return s.DB.Where("id = ?", id).Assign(model.UserIntelligenceLabel{ID: id, TenantID: tenantID, Fingerprint: fingerprint, Label: label, Source: source, CreatedAt: time.Now()}).FirstOrCreate(&model.UserIntelligenceLabel{}).Error
}

func (s *UserIntelligenceService) TrainingReadiness(tenantID, label string) (map[string]any, error) {
	var positive, total int64
	q := s.DB.Model(&model.UserIntelligenceLabel{}).Where("tenant_id = ? AND label = ?", tenantID, label)
	if err := q.Count(&positive).Error; err != nil {
		return nil, err
	}
	if err := s.DB.Model(&model.UserIntelligenceLabel{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		return nil, err
	}
	return map[string]any{"tenant_id": tenantID, "label": label, "examples": total, "positive": positive, "negative": total - positive, "ready": total >= 20 && positive >= 5 && total-positive >= 5}, nil
}

type UserIntelligenceSnapshot struct {
	TenantID             string             `json:"tenant_id"`
	Fingerprint          string             `json:"fingerprint"`
	IP                   string             `json:"ip"`
	UserAgent            string             `json:"user_agent"`
	Tags                 []string           `json:"tags"`
	Scores               map[string]float64 `json:"scores"`
	Features             map[string]float64 `json:"features"`
	AnomalyScore         float64            `json:"anomaly_score"`
	PredictedNextEvent   string             `json:"predicted_next_event"`
	PredictedEventsToday int64              `json:"predicted_events_today"`
	Confidence           float64            `json:"confidence"`
	ComputedAt           time.Time          `json:"computed_at"`
}

func UserFingerprint(ip, userAgent string) string {
	sum := sha256.Sum256([]byte(ip + "\x00" + userAgent))
	return hex.EncodeToString(sum[:])
}

func (s *UserIntelligenceService) Refresh(tenantID string, now time.Time) (int, error) {
	var sessions []model.Session
	if err := s.DB.Where("tenant_id = ?", tenantID).Find(&sessions).Error; err != nil {
		return 0, err
	}
	if len(sessions) == 0 {
		return 0, nil
	}
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.ID)
	}
	var events []model.Event
	if err := s.DB.Where("session_id IN ? AND is_aggregated = ?", ids, false).Find(&events).Error; err != nil {
		return 0, err
	}
	type aggregate struct {
		sessionCount, eventCount int64
		lastSeen                 time.Time
		names                    map[string]int64
		timestamps               []int64
		ip, ua                   string
	}
	groups := map[string]*aggregate{}
	sessionKeys := map[string]string{}
	for _, session := range sessions {
		key := session.IP + "\x00" + session.UserAgent
		sessionKeys[session.ID] = key
		if groups[key] == nil {
			groups[key] = &aggregate{names: map[string]int64{}, ip: session.IP, ua: session.UserAgent}
		}
		g := groups[key]
		g.sessionCount++
		if session.LastSeen.After(g.lastSeen) {
			g.lastSeen = session.LastSeen
		}
	}
	for _, event := range events {
		if key, ok := sessionKeys[event.SessionID]; ok {
			g := groups[key]
			g.eventCount++
			g.names[event.EventName]++
			g.timestamps = append(g.timestamps, event.ClientTS)
		}
	}
	count := 0
	for key, g := range groups {
		tags := []string{"new_user"}
		scores := map[string]float64{}
		features := map[string]float64{"sessions": float64(g.sessionCount), "events": float64(g.eventCount)}
		if g.sessionCount > 1 {
			tags = append(tags, "returning_user")
		}
		if strings.Contains(strings.ToLower(g.ua), "bot") || strings.Contains(strings.ToLower(g.ua), "crawler") || strings.Contains(strings.ToLower(g.ua), "spider") {
			tags = append(tags, "bot", "automation")
			scores["bot"] = 1
		}
		buyer, intent := false, 0.0
		for name, n := range g.names {
			lower := strings.ToLower(name)
			if strings.Contains(lower, "purchase") || strings.Contains(lower, "checkout") || strings.Contains(lower, "payment") || strings.Contains(lower, "order_completed") {
				buyer = true
			}
			if strings.Contains(lower, "product") || strings.Contains(lower, "cart") || strings.Contains(lower, "checkout") {
				intent += float64(n)
			}
		}
		if buyer {
			tags = append(tags, "buyer")
			scores["buyer"] = 1
		} else if intent > 0 {
			tags = append(tags, "potential_buyer")
			scores["buyer"] = math.Min(0.95, intent/10)
		}
		if intent >= 2 {
			tags = append(tags, "high_intent")
		}
		if !g.lastSeen.IsZero() && g.lastSeen.Before(now.Add(-30*24*time.Hour)) {
			tags = append(tags, "inactive")
		}
		anomaly := 0.0
		if g.eventCount >= 1000 && g.sessionCount <= 2 {
			anomaly = math.Min(1, float64(g.eventCount)/10000)
			tags = append(tags, "anomalous")
		}
		if len(g.timestamps) > 0 {
			sort.Slice(g.timestamps, func(i, j int) bool { return g.timestamps[i] < g.timestamps[j] })
		}
		next := ""
		best := int64(0)
		for name, n := range g.names {
			if n > best {
				best, next = n, name
			}
		}
		predicted := int64(math.Round(float64(g.eventCount) * 2))
		confidence := math.Min(1, float64(g.eventCount)/10)
		row := model.UserIntelligence{ID: tenantID + ":" + UserFingerprint(g.ip, g.ua), TenantID: tenantID, Fingerprint: UserFingerprint(g.ip, g.ua), IP: g.ip, UserAgent: g.ua, AnomalyScore: anomaly, PredictedNextEvent: next, PredictedEventsToday: predicted, Confidence: confidence, ComputedAt: now}
		encodedTags, _ := json.Marshal(tags)
		encodedScores, _ := json.Marshal(scores)
		encodedFeatures, _ := json.Marshal(features)
		row.Tags, row.Scores, row.Features = string(encodedTags), string(encodedScores), string(encodedFeatures)
		if err := s.DB.Where("id = ?", row.ID).Assign(row).FirstOrCreate(&row).Error; err != nil {
			return count, err
		}
		count++
		_ = key
	}
	return count, nil
}

func (s *UserIntelligenceService) List(tenantID string, limit int) ([]UserIntelligenceSnapshot, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var rows []model.UserIntelligence
	if err := s.DB.Where("tenant_id = ?", tenantID).Order("computed_at desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]UserIntelligenceSnapshot, 0, len(rows))
	for _, row := range rows {
		var snapshot UserIntelligenceSnapshot
		snapshot.TenantID, snapshot.Fingerprint, snapshot.IP, snapshot.UserAgent = row.TenantID, row.Fingerprint, row.IP, row.UserAgent
		snapshot.AnomalyScore, snapshot.PredictedNextEvent, snapshot.PredictedEventsToday, snapshot.Confidence, snapshot.ComputedAt = row.AnomalyScore, row.PredictedNextEvent, row.PredictedEventsToday, row.Confidence, row.ComputedAt
		_ = json.Unmarshal([]byte(row.Tags), &snapshot.Tags)
		_ = json.Unmarshal([]byte(row.Scores), &snapshot.Scores)
		_ = json.Unmarshal([]byte(row.Features), &snapshot.Features)
		out = append(out, snapshot)
	}
	return out, nil
}
