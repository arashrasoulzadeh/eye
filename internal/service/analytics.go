package service

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// AnalyticsService computes funnel conversion and retention reports
// directly from raw events — there's no materialized read model for these
// yet (unlike AggregateCount for simple counts), so both scan Event rows
// for the requested tenant/time range on each call.
type analyticsFunnelCacheEntry struct {
	expires time.Time
	value   []FunnelStepResult
}

type analyticsSuggestedCacheEntry struct {
	expires time.Time
	value   []SuggestedFunnelStep
}

type AnalyticsService struct {
	DB             *gorm.DB
	cacheMu        sync.Mutex
	funnelCache    map[string]analyticsFunnelCacheEntry
	suggestedCache map[string]analyticsSuggestedCacheEntry
}

type SessionEventCount struct {
	SessionID string `json:"session_id"`
	Count     int    `json:"count"`
}

type DeviceBreakdownRow struct {
	Platform string `json:"platform"`
	Sessions int    `json:"sessions"`
}

type GeoBreakdownRow struct {
	Location string `json:"location"`
	Sessions int    `json:"sessions"`
}

func (a *AnalyticsService) GeoBreakdown(tenantID, dimension string) ([]GeoBreakdownRow, error) {
	if dimension != "country" && dimension != "region" && dimension != "city" {
		return nil, fmt.Errorf("dimension must be country, region, or city")
	}
	var sessions []model.Session
	if err := a.DB.Where("tenant_id = ?", tenantID).Find(&sessions).Error; err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, session := range sessions {
		location := map[string]string{"country": session.Country, "region": session.Region, "city": session.City}[dimension]
		if location != "" {
			counts[location]++
			continue
		}
		var device map[string]any
		_ = json.Unmarshal([]byte(session.DeviceInfo), &device)
		legacyLocation, _ := device[dimension].(string)
		if legacyLocation == "" {
			legacyLocation = "unknown"
		}
		counts[legacyLocation]++
	}
	rows := make([]GeoBreakdownRow, 0, len(counts))
	for location, count := range counts {
		rows = append(rows, GeoBreakdownRow{Location: location, Sessions: count})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Location < rows[j].Location })
	return rows, nil
}

type TopNRow struct {
	EventName string `json:"event_name"`
	DimsKey   string `json:"dims_key"`
	Count     int64  `json:"count"`
}

type WindowCount struct {
	WindowStart int64 `json:"window_start"`
	Count       int   `json:"count"`
}

type PercentileSummary struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

func (a *AnalyticsService) SessionEventPercentiles(tenantID string) (PercentileSummary, error) {
	rows, err := a.EventsPerSession(tenantID)
	if err != nil {
		return PercentileSummary{}, err
	}
	if len(rows) == 0 {
		return PercentileSummary{}, nil
	}
	counts := make([]int, len(rows))
	for i, row := range rows {
		counts[i] = row.Count
	}
	sort.Ints(counts)
	percentile := func(p float64) float64 {
		position := p * float64(len(counts)-1)
		low := int(position)
		high := low
		if high+1 < len(counts) {
			high++
		}
		fraction := position - float64(low)
		return float64(counts[low]) + fraction*float64(counts[high]-counts[low])
	}
	return PercentileSummary{P50: percentile(0.50), P95: percentile(0.95), P99: percentile(0.99)}, nil
}

// DistinctSessions estimates unique sessions with a compact 64-register
// HyperLogLog sketch. It is intentionally rebuilt for the requested tenant
// and event name so the result cannot cross tenant boundaries.
func (a *AnalyticsService) DistinctSessions(tenantID, eventName string) (float64, error) {
	var sessions []string
	query := a.DB.Model(&model.Event{}).Distinct("session_id").Where("tenant_id = ?", tenantID)
	if eventName != "" {
		query = query.Where("event_name = ?", eventName)
	}
	if err := query.Pluck("session_id", &sessions).Error; err != nil {
		return 0, err
	}
	const registers = 64
	var values [registers]uint8
	for _, session := range sessions {
		hash := sha256.Sum256([]byte(session))
		index := int(hash[0] & (registers - 1))
		var rank uint8 = 1
		for _, b := range hash[1:] {
			if b == 0 {
				rank += 8
				continue
			}
			for mask := byte(0x80); mask&b == 0; mask >>= 1 {
				rank++
			}
			break
		}
		if rank > values[index] {
			values[index] = rank
		}
	}
	var sum float64
	zeroes := 0
	for _, value := range values {
		sum += 1 / float64(uint64(1)<<value)
		if value == 0 {
			zeroes++
		}
	}
	estimate := 0.7213 * registers * registers / sum
	if estimate <= 2.5*registers && zeroes > 0 {
		estimate = registers * math.Log(float64(registers)/float64(zeroes))
	}
	return estimate, nil
}

func (a *AnalyticsService) SlidingWindowCounts(tenantID, eventName string, windowSeconds int64, fromTS, toTS *int64) ([]WindowCount, error) {
	if windowSeconds <= 0 {
		return nil, fmt.Errorf("window_seconds must be positive")
	}
	query := a.DB.Model(&model.Event{}).Where("tenant_id = ? AND event_name = ?", tenantID, eventName)
	if fromTS != nil {
		query = query.Where("client_ts >= ?", *fromTS)
	}
	if toTS != nil {
		query = query.Where("client_ts <= ?", *toTS)
	}
	var events []model.Event
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	counts := map[int64]int{}
	for _, event := range events {
		start := event.ClientTS / (windowSeconds * 1000) * (windowSeconds * 1000)
		counts[start]++
	}
	rows := make([]WindowCount, 0, len(counts))
	for start, count := range counts {
		rows = append(rows, WindowCount{WindowStart: start, Count: count})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].WindowStart < rows[j].WindowStart })
	return rows, nil
}

func (a *AnalyticsService) TopN(tenantID, eventName string, limit int) ([]TopNRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	var rows []TopNRow
	query := a.DB.Model(&model.AggregateCount{}).Select("event_name, dims_key, count").Where("tenant_id = ?", tenantID)
	if eventName != "" {
		query = query.Where("event_name = ?", eventName)
	}
	if err := query.Order("count desc").Order("event_name asc").Order("dims_key asc").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (a *AnalyticsService) DeviceBreakdown(tenantID string) ([]DeviceBreakdownRow, error) {
	var sessions []model.Session
	if err := a.DB.Where("tenant_id = ?", tenantID).Find(&sessions).Error; err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, session := range sessions {
		var device map[string]any
		_ = json.Unmarshal([]byte(session.DeviceInfo), &device)
		platform, _ := device["platform"].(string)
		if platform == "" {
			platform = "unknown"
		}
		counts[platform]++
	}
	rows := make([]DeviceBreakdownRow, 0, len(counts))
	for platform, count := range counts {
		rows = append(rows, DeviceBreakdownRow{Platform: platform, Sessions: count})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Platform < rows[j].Platform })
	return rows, nil
}

// EventsPerSession returns event counts scoped to one tenant, ordered from
// busiest session to least busy. It intentionally counts raw events so the
// result reflects the same data visible to query consumers.
func (a *AnalyticsService) EventsPerSession(tenantID string) ([]SessionEventCount, error) {
	var events []model.Event
	if err := a.DB.Model(&model.Event{}).Where("tenant_id = ?", tenantID).Find(&events).Error; err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, event := range events {
		counts[event.SessionID]++
	}
	result := make([]SessionEventCount, 0, len(counts))
	for sessionID, count := range counts {
		result = append(result, SessionEventCount{SessionID: sessionID, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].SessionID < result[j].SessionID
		}
		return result[i].Count > result[j].Count
	})
	return result, nil
}

// FunnelStepResult is one step of a funnel: how many distinct sessions
// reached it (in order, i.e. only counting sessions that already reached
// every prior step), and the conversion rate from the previous step.
type FunnelStepResult struct {
	EventName      string  `json:"event_name"`
	Sessions       int     `json:"sessions"`
	ConversionRate float64 `json:"conversion_rate"` // vs. the previous step; 1.0 for the first step
}

// SuggestedFunnelStep is a human-readable stage inferred from the most common
// ordered event path in a tenant's sessions.
type SuggestedFunnelStep struct {
	Position  int    `json:"position"`
	EventName string `json:"event_name"`
	Title     string `json:"title"`
	Sessions  int    `json:"sessions"`
}

// SuggestedFunnel discovers a useful starting funnel without requiring the
// operator to guess event names. It chooses the most common first event, then
// greedily follows the most common next distinct event across sessions.
func (a *AnalyticsService) SuggestedFunnel(tenantID string, fromTS, toTS *int64) ([]SuggestedFunnelStep, error) {
	key := fmt.Sprintf("%s|%v|%v", tenantID, fromTS, toTS)
	a.cacheMu.Lock()
	if entry, ok := a.suggestedCache[key]; ok && time.Now().Before(entry.expires) {
		value := append([]SuggestedFunnelStep(nil), entry.value...)
		a.cacheMu.Unlock()
		return value, nil
	}
	a.cacheMu.Unlock()
	q := a.DB.Where("is_aggregated = ?", false)
	if tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if fromTS != nil {
		q = q.Where("client_ts >= ?", *fromTS)
	}
	if toTS != nil {
		q = q.Where("client_ts <= ?", *toTS)
	}
	var events []model.Event
	if err := q.Order("session_id asc, client_ts asc").Find(&events).Error; err != nil {
		return nil, err
	}
	sessions := map[string][]model.Event{}
	for _, event := range events {
		sessions[event.SessionID] = append(sessions[event.SessionID], event)
	}
	firstCounts := map[string]int{}
	for _, path := range sessions {
		if len(path) > 0 {
			firstCounts[path[0].EventName]++
		}
	}
	current := mostCommonEvent(firstCounts)
	if current == "" {
		return a.cacheSuggested(key, []SuggestedFunnelStep{}), nil
	}
	result := make([]SuggestedFunnelStep, 0, 5)
	remaining := map[string]bool{}
	for len(result) < 5 && current != "" {
		count := 0
		nextCounts := map[string]int{}
		for sessionID, path := range sessions {
			_ = sessionID
			for i, event := range path {
				if event.EventName != current {
					continue
				}
				if len(result) == 0 && i != 0 {
					continue
				}
				if len(result) > 0 && !remaining[sessionID] {
					continue
				}
				count++
				for j := i + 1; j < len(path); j++ {
					if path[j].EventName != current {
						nextCounts[path[j].EventName]++
						break
					}
				}
				break
			}
		}
		result = append(result, SuggestedFunnelStep{Position: len(result) + 1, EventName: current, Title: humanEventTitle(current), Sessions: count})
		if len(result) == 5 {
			break
		}
		next := mostCommonEvent(nextCounts)
		if next == "" {
			break
		}
		remaining = map[string]bool{}
		for sessionID, path := range sessions {
			for i, event := range path {
				if event.EventName == current {
					for j := i + 1; j < len(path); j++ {
						if path[j].EventName == next {
							remaining[sessionID] = true
						}
						break
					}
					break
				}
			}
		}
		current = next
	}
	return a.cacheSuggested(key, result), nil
}

func (a *AnalyticsService) cacheSuggested(key string, value []SuggestedFunnelStep) []SuggestedFunnelStep {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	if a.suggestedCache == nil {
		a.suggestedCache = make(map[string]analyticsSuggestedCacheEntry)
	}
	if len(a.suggestedCache) >= 128 {
		for oldKey := range a.suggestedCache {
			delete(a.suggestedCache, oldKey)
			break
		}
	}
	a.suggestedCache[key] = analyticsSuggestedCacheEntry{expires: time.Now().Add(10 * time.Second), value: append([]SuggestedFunnelStep(nil), value...)}
	return append([]SuggestedFunnelStep(nil), value...)
}

func mostCommonEvent(counts map[string]int) string {
	best := ""
	bestCount := 0
	for name, count := range counts {
		if count > bestCount || (count == bestCount && name < best) {
			best, bestCount = name, count
		}
	}
	return best
}

func humanEventTitle(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == '-' || r == '.' })
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

// Funnel computes, for the given ordered steps, how many distinct sessions
// reached each step in order: a session counts at step N only if it has an
// event for step N-1 at an earlier-or-equal ClientTS than the event that
// qualifies it for step N, and it already counted at step N-1. Sessions
// that fire the steps out of order, or skip one, stop counting at the last
// step they genuinely reached in sequence.
func (a *AnalyticsService) Funnel(tenantID string, steps []string, fromTS, toTS *int64) ([]FunnelStepResult, error) {
	if len(steps) == 0 {
		return nil, nil
	}
	key := fmt.Sprintf("%s|%s|%v|%v", tenantID, strings.Join(steps, "\x00"), fromTS, toTS)
	a.cacheMu.Lock()
	if entry, ok := a.funnelCache[key]; ok && time.Now().Before(entry.expires) {
		value := append([]FunnelStepResult(nil), entry.value...)
		a.cacheMu.Unlock()
		return value, nil
	}
	a.cacheMu.Unlock()

	query := a.DB.Model(&model.Event{}).Where("is_aggregated = ? AND event_name IN ?", false, steps)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if fromTS != nil {
		query = query.Where("client_ts >= ?", *fromTS)
	}
	if toTS != nil {
		query = query.Where("client_ts <= ?", *toTS)
	}
	var events []model.Event
	if err := query.Order("client_ts asc").Find(&events).Error; err != nil {
		return nil, err
	}

	// sessionID -> timestamp at which it last advanced to this step index.
	progress := map[string]int{} // sessionID -> index of furthest step reached + 1 (0 = none yet)
	lastTS := map[string]int64{} // sessionID -> ClientTS of the event that satisfied `progress`
	stepIndex := map[string]int{}
	for i, s := range steps {
		stepIndex[s] = i
	}

	counts := make([]int, len(steps))
	for _, e := range events {
		idx, ok := stepIndex[e.EventName]
		if !ok {
			continue
		}
		cur := progress[e.SessionID]
		if idx == cur && e.ClientTS >= lastTS[e.SessionID] {
			progress[e.SessionID] = cur + 1
			lastTS[e.SessionID] = e.ClientTS
			counts[idx]++
		}
	}

	results := make([]FunnelStepResult, len(steps))
	for i, s := range steps {
		rate := 1.0
		if i > 0 && counts[i-1] > 0 {
			rate = float64(counts[i]) / float64(counts[i-1])
		} else if i > 0 {
			rate = 0
		}
		results[i] = FunnelStepResult{EventName: s, Sessions: counts[i], ConversionRate: rate}
	}
	a.cacheMu.Lock()
	if a.funnelCache == nil {
		a.funnelCache = make(map[string]analyticsFunnelCacheEntry)
	}
	if len(a.funnelCache) >= 128 {
		for oldKey := range a.funnelCache {
			delete(a.funnelCache, oldKey)
			break
		}
	}
	a.funnelCache[key] = analyticsFunnelCacheEntry{expires: time.Now().Add(10 * time.Second), value: append([]FunnelStepResult(nil), results...)}
	a.cacheMu.Unlock()
	return results, nil
}

// RetentionRow is one cohort's activity: the day (or week) its sessions
// first appeared, how many sessions started that day, and how many of
// those same sessions were still active N periods later.
type RetentionRow struct {
	CohortPeriod string    `json:"cohort_period"` // e.g. "2026-11-03" or "2026-W45"
	CohortSize   int       `json:"cohort_size"`
	ActiveByN    []int     `json:"active_by_n"`  // index N = sessions from this cohort active N periods later (N=0 is the cohort period itself)
	RetainedPct  []float64 `json:"retained_pct"` // ActiveByN[n] / CohortSize
}

// Retention computes a classic retention matrix for a tenant: group
// sessions into cohorts by the period (day or week) their first event
// occurred, then for each cohort and each later period count how many of
// that cohort's sessions had at least one event in that period.
func (a *AnalyticsService) Retention(tenantID, granularity string, bounds ...*int64) ([]RetentionRow, error) {
	var events []model.Event
	db := a.DB.Model(&model.Event{})
	if tenantID != "" {
		db = db.Where("tenant_id = ?", tenantID)
	}
	if len(bounds) > 0 && bounds[0] != nil {
		db = db.Where("client_ts >= ?", *bounds[0])
	}
	if len(bounds) > 1 && bounds[1] != nil {
		db = db.Where("client_ts <= ?", *bounds[1])
	}
	if err := db.Order("client_ts asc").Find(&events).Error; err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, nil
	}

	periodOf := dayPeriod
	periodIndex := dayIndex
	if granularity == "weekly" {
		periodOf = weekPeriod
		periodIndex = weekIndex
	}

	firstSeen := map[string]int64{}              // sessionID -> first ClientTS
	activePeriods := map[string]map[int64]bool{} // sessionID -> set of period-index they were active in
	for _, e := range events {
		if _, ok := firstSeen[e.SessionID]; !ok {
			firstSeen[e.SessionID] = e.ClientTS
		}
		idx := periodIndex(e.ClientTS)
		if activePeriods[e.SessionID] == nil {
			activePeriods[e.SessionID] = map[int64]bool{}
		}
		activePeriods[e.SessionID][idx] = true
	}

	// group sessions into cohorts by their first-seen period
	cohortSessions := map[string][]string{} // period label -> session IDs
	cohortStartIdx := map[string]int64{}
	for sid, ts := range firstSeen {
		label := periodOf(ts)
		cohortSessions[label] = append(cohortSessions[label], sid)
		cohortStartIdx[label] = periodIndex(ts)
	}

	// max periods forward we need to report: furthest period index seen
	// minus the earliest cohort start (so every row's ActiveByN is the same
	// length, with trailing zeros past that cohort's own data).
	var maxPeriodIdx, minCohortStart int64
	first := true
	for _, e := range events {
		idx := periodIndex(e.ClientTS)
		if first || idx > maxPeriodIdx {
			maxPeriodIdx = idx
		}
		first = false
	}
	first = true
	for _, start := range cohortStartIdx {
		if first || start < minCohortStart {
			minCohortStart = start
		}
		first = false
	}
	maxN := int(maxPeriodIdx - minCohortStart)
	if maxN < 0 {
		maxN = 0
	}

	labels := make([]string, 0, len(cohortSessions))
	for label := range cohortSessions {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	rows := make([]RetentionRow, 0, len(labels))
	for _, label := range labels {
		sessions := cohortSessions[label]
		start := cohortStartIdx[label]
		activeByN := make([]int, maxN+1)
		retainedPct := make([]float64, maxN+1)
		for n := 0; n <= maxN; n++ {
			count := 0
			for _, sid := range sessions {
				if activePeriods[sid][start+int64(n)] {
					count++
				}
			}
			activeByN[n] = count
			retainedPct[n] = float64(count) / float64(len(sessions))
		}
		rows = append(rows, RetentionRow{
			CohortPeriod: label,
			CohortSize:   len(sessions),
			ActiveByN:    activeByN,
			RetainedPct:  retainedPct,
		})
	}
	return rows, nil
}

const daySeconds = 86400
const weekSeconds = 7 * daySeconds

func dayIndex(clientTSMillis int64) int64  { return clientTSMillis / 1000 / daySeconds }
func weekIndex(clientTSMillis int64) int64 { return clientTSMillis / 1000 / weekSeconds }

func timeFromMillis(clientTSMillis int64) time.Time {
	return time.UnixMilli(clientTSMillis).UTC()
}

func dayPeriod(clientTSMillis int64) string {
	return timeFromMillis(clientTSMillis).Format("2006-01-02")
}

func weekPeriod(clientTSMillis int64) string {
	y, w := timeFromMillis(clientTSMillis).ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}
