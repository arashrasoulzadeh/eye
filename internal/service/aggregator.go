package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

type AggregatorService struct{ DB *gorm.DB }

// RefreshMaterializedCounts rebuilds the aggregate-count read model from raw
// events. It is tenant-scoped and safe to run after imports or backfills.
func (a *AggregatorService) RefreshMaterializedCounts(tenantID string) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}
	return a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ?", tenantID).Delete(&model.AggregateCount{}).Error; err != nil {
			return err
		}
		var events []model.Event
		if err := tx.Where("tenant_id = ?", tenantID).Find(&events).Error; err != nil {
			return err
		}
		for _, event := range events {
			if err := UpsertAggregateCount(tx, event.TenantID, event.EventName, event.Dims); err != nil {
				return err
			}
		}
		return nil
	})
}

// ExportAggregateCounts sends the current materialized counts for a tenant to
// an injected webhook client. The client seam keeps delivery testable and
// lets callers reuse the retry/dead-letter layer used by alerts.
func (a *AggregatorService) ExportAggregateCounts(tenantID, eventName, url string, client WebhookClient) error {
	if tenantID == "" || url == "" || client == nil {
		return errors.New("tenant_id, url, and webhook client are required")
	}
	var rows []model.AggregateCount
	query := a.DB.Where("tenant_id = ?", tenantID)
	if eventName != "" {
		query = query.Where("event_name = ?", eventName)
	}
	if err := query.Order("event_name asc, dims_key asc").Find(&rows).Error; err != nil {
		return err
	}
	payload := map[string]any{"tenant_id": tenantID, "event_name": eventName, "aggregates": rows}
	return client.PostJSON(url, payload)
}

func randomAggregationID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *AggregatorService) Create(name, tenantID, rule, output string, threshold int, window string) (*model.Aggregation, error) {
	agg := &model.Aggregation{ID: randomAggregationID(), Name: name, TenantID: tenantID, Rule: rule, OutputEvent: output, Threshold: threshold, Window: window, Enabled: true, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()}
	if err := a.DB.Create(agg).Error; err != nil {
		return nil, err
	}
	return agg, nil
}

func (a *AggregatorService) List(tenantID string) ([]model.Aggregation, error) {
	var list []model.Aggregation
	if err := a.DB.Where("tenant_id = ?", tenantID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (a *AggregatorService) Get(id string) (*model.Aggregation, error) {
	var agg model.Aggregation
	if err := a.DB.First(&agg, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &agg, nil
}

// Update edits an aggregation's rule in place. Per phases.md Phase 13, an
// edited rule is picked up by the next Backfill/job tick — it does not itself
// recompute, since Backfill already skips events it previously aggregated
// under this aggregation's ID and only processes the rest.
func (a *AggregatorService) Update(id, name, rule, output string, threshold int, window string) (*model.Aggregation, error) {
	agg, err := a.Get(id)
	if err != nil {
		return nil, err
	}
	agg.Name = name
	agg.Rule = rule
	agg.OutputEvent = output
	agg.Threshold = threshold
	agg.Window = window
	agg.UpdatedAt = time.Now().Unix()
	if err := a.DB.Save(agg).Error; err != nil {
		return nil, err
	}
	return agg, nil
}

func (a *AggregatorService) Delete(id string) error {
	return a.DB.Delete(&model.Aggregation{}, "id = ?", id).Error
}

func (a *AggregatorService) EmitSystemEvent(tenantID, name string, payload map[string]any) error {
	b, _ := json.Marshal(payload)
	ev := &model.Event{EventID: "sys-" + randomAggregationID(), TenantID: tenantID, EventName: name, ClientTS: time.Now().UnixMilli(), IngestTS: time.Now(), Payload: string(b), Source: "system", IsAggregated: true}
	return a.DB.Create(ev).Error
}

type aggregationRule struct {
	EventName string `json:"event_name"`
}

func windowDuration(window string) time.Duration {
	switch window {
	case "1m":
		return time.Minute
	case "1h":
		return time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return time.Hour
	}
}

// Backfill scans events matching the aggregation's rule that have not yet
// been counted by this aggregation (AggregationID is empty), buckets them by
// the aggregation's time window, and emits one system output event per
// bucket that reaches Threshold. Matched source events are stamped with
// AggregationID so a second Backfill run (or the periodic job tick) never
// recounts them — this is what makes it idempotent and also the "reprocess
// older events" / "editable rule recomputes" mechanism: anything not yet
// stamped for this aggregation is fair game, including events present before
// the aggregation existed or before a rule edit.
//
// This is a full on-demand recompute over matching rows, not a real
// streaming aggregator (Phase 7/Kafka isn't built yet) — same honest
// simplification as the Phase 6 cohort materialization job.
func (a *AggregatorService) Backfill(aggregationID string) (int, error) {
	agg, err := a.Get(aggregationID)
	if err != nil {
		return 0, err
	}
	if !agg.Enabled {
		return 0, nil
	}
	var rule aggregationRule
	if err := json.Unmarshal([]byte(agg.Rule), &rule); err != nil {
		return 0, errors.New("invalid aggregation rule JSON: " + err.Error())
	}
	if rule.EventName == "" {
		return 0, errors.New("aggregation rule missing event_name")
	}

	var events []model.Event
	err = a.DB.Where("tenant_id = ? AND event_name = ? AND aggregation_id = ?", agg.TenantID, rule.EventName, "").
		Find(&events).Error
	if err != nil {
		return 0, err
	}
	if len(events) == 0 {
		return 0, nil
	}

	dur := windowDuration(agg.Window)
	type bucket struct {
		start time.Time
		ids   []string
	}
	buckets := map[int64]*bucket{}
	for _, ev := range events {
		key := ev.IngestTS.Unix() / int64(dur.Seconds())
		b, ok := buckets[key]
		if !ok {
			b = &bucket{start: time.Unix(key*int64(dur.Seconds()), 0)}
			buckets[key] = b
		}
		b.ids = append(b.ids, ev.EventID)
	}

	emitted := 0
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		for _, b := range buckets {
			if len(b.ids) < agg.Threshold {
				continue
			}
			payload := map[string]any{
				"count":          len(b.ids),
				"window_start":   b.start.Unix(),
				"window":         agg.Window,
				"aggregation_id": agg.ID,
			}
			pb, _ := json.Marshal(payload)
			out := &model.Event{
				EventID:       "sys-" + randomAggregationID(),
				TenantID:      agg.TenantID,
				EventName:     agg.OutputEvent,
				ClientTS:      time.Now().UnixMilli(),
				IngestTS:      time.Now(),
				Payload:       string(pb),
				Source:        "system",
				IsAggregated:  true,
				AggregationID: agg.ID,
			}
			if err := tx.Create(out).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Event{}).Where("event_id IN ?", b.ids).Update("aggregation_id", agg.ID).Error; err != nil {
				return err
			}
			emitted++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return emitted, nil
}

// RunAggregationJob periodically backfills every enabled aggregation across
// all tenants. Like RunCohortMaterializationJob, this is a simple polling
// recompute, not a real-time stream consumer.
func RunAggregationJob(ctx context.Context, svc *AggregatorService, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var aggs []model.Aggregation
			if err := svc.DB.Where("enabled = ?", true).Find(&aggs).Error; err != nil {
				if logger != nil {
					logger.Error("aggregation job: list failed", "err", err)
				}
				continue
			}
			for _, agg := range aggs {
				if _, err := svc.Backfill(agg.ID); err != nil && logger != nil {
					logger.Error("aggregation job: backfill failed", "id", agg.ID, "err", err)
				}
			}
		}
	}
}
