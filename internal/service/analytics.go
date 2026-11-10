package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// AnalyticsService computes funnel conversion and retention reports
// directly from raw events — there's no materialized read model for these
// yet (unlike AggregateCount for simple counts), so both scan Event rows
// for the requested tenant/time range on each call.
type AnalyticsService struct{ DB *gorm.DB }

type SessionEventCount struct {
	SessionID string `json:"session_id"`
	Count     int    `json:"count"`
}

type DeviceBreakdownRow struct {
	Platform string `json:"platform"`
	Sessions int    `json:"sessions"`
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

	query := a.DB.Model(&model.Event{}).Where("tenant_id = ?", tenantID).Where("event_name IN ?", steps)
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
func (a *AnalyticsService) Retention(tenantID, granularity string) ([]RetentionRow, error) {
	var events []model.Event
	if err := a.DB.Model(&model.Event{}).Where("tenant_id = ?", tenantID).Order("client_ts asc").Find(&events).Error; err != nil {
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
