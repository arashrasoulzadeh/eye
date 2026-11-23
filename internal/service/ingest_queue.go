package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ingestQueueBatchSize = 10

// IngestQueue uses the existing pending-event table as a durable buffer.
type IngestQueue struct {
	DB               *gorm.DB
	SetTenantContext func(*gorm.DB, string) error
	Sessions         *LiveSessionHub
	depth            atomic.Int64
}

func (q *IngestQueue) Depth() int { return int(q.depth.Load()) }

func (q *IngestQueue) Enqueue(events []*model.Event, aggregate bool) ([]*model.Event, error) {
	accepted := make([]*model.Event, 0, len(events))
	err := q.DB.Transaction(func(tx *gorm.DB) error {
		for _, event := range events {
			payload, err := json.Marshal(event)
			if err != nil {
				return err
			}
			row := model.PendingStreamEvent{EventID: event.EventID, Payload: string(payload), SkipAggregate: !aggregate}
			if event.Source != "replay" && event.ContentHash != "" {
				row.ContentHash = &event.ContentHash
			}
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				accepted = append(accepted, event)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return accepted, nil
}

func (q *IngestQueue) Run(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := q.Drain(ctx); err != nil && ctx.Err() == nil {
				logger.Error("ingest queue drain failed", "err", err)
			}
		}
	}
}

func (q *IngestQueue) Drain(ctx context.Context) error {
	var depth int64
	if err := q.DB.WithContext(ctx).Model(&model.PendingStreamEvent{}).Where("decode_error IS NULL").Count(&depth).Error; err != nil {
		return err
	}
	q.depth.Store(depth)
	for remaining := (depth + 9) / 10; remaining > 0; {
		processed, err := q.drainBatch(ctx, int(min(remaining, ingestQueueBatchSize)))
		if err != nil {
			return err
		}
		if processed == 0 {
			break
		}
		remaining -= int64(processed)
		q.depth.Add(-int64(processed))
	}
	return nil
}

func (q *IngestQueue) drainBatch(ctx context.Context, limit int) (int, error) {
	var rows []model.PendingStreamEvent
	type notification struct {
		event   *model.Event
		session *model.Session
	}
	var notifications []notification
	err := q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("decode_error IS NULL").Order("created_at, event_id").Limit(limit).Find(&rows).Error; err != nil {
			return err
		}
		ids := make([]string, 0, len(rows))
		events := make([]*model.Event, 0, len(rows))
		skipAggregate := make([]bool, 0, len(rows))
		for _, row := range rows {
			var event model.Event
			decodeErr := json.Unmarshal([]byte(row.Payload), &event)
			if decodeErr != nil || event.EventID != row.EventID || event.EventID == "" || event.TenantID == "" {
				// Keep malformed legacy rows for inspection while allowing
				// later events to drain. Avoid recording payload contents.
				err := tx.Model(&row).Update("decode_error", "invalid pending event payload or identity").Error
				if err != nil {
					return err
				}
				continue
			}
			events = append(events, &event)
			skipAggregate = append(skipAggregate, row.SkipAggregate)
			ids = append(ids, row.EventID)
		}
		var newEvents []*model.Event
		for start := 0; start < len(events); {
			end := start + 1
			for end < len(events) && events[end].TenantID == events[start].TenantID {
				end++
			}
			inserted, err := q.insertEvents(tx, events[start:end])
			if err != nil {
				return err
			}
			for i := start; i < end; i++ {
				event := events[i]
				if !inserted[event.EventID] {
					continue
				}
				newEvents = append(newEvents, event)
				session, err := updateQueuedEventCounts(tx, event, skipAggregate[i])
				if err != nil {
					return fmt.Errorf("update counts for pending event %q: %w", event.EventID, err)
				}
				if session != nil {
					notifications = append(notifications, notification{event, session})
				}
			}
			start = end
		}
		if len(ids) == 0 {
			return nil
		}
		if err := UpsertEventBuckets(tx, newEvents...); err != nil {
			return err
		}
		return tx.Delete(&model.PendingStreamEvent{}, "event_id IN ?", ids).Error
	})
	if err != nil {
		return 0, err
	}
	if q.Sessions != nil {
		for _, item := range notifications {
			q.Sessions.Update(item.session)
			q.Sessions.AddEvent(item.event.SessionID, item.event)
		}
	}
	return len(rows), nil
}

// insertEvents scopes each multi-row insert to one tenant. Already-persisted
// IDs can be present in the pending table after upgrading from the old worker.
func (q *IngestQueue) insertEvents(tx *gorm.DB, events []*model.Event) (map[string]bool, error) {
	if q.SetTenantContext != nil {
		if err := q.SetTenantContext(tx, events[0].TenantID); err != nil {
			return nil, err
		}
	}
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.EventID)
	}
	var existing []string
	if err := tx.Model(&model.Event{}).Where("event_id IN ?", ids).Pluck("event_id", &existing).Error; err != nil {
		return nil, err
	}
	inserted := make(map[string]bool, len(events))
	for _, id := range existing {
		inserted[id] = false
	}
	newEvents := make([]*model.Event, 0, len(events)-len(existing))
	for _, event := range events {
		if _, exists := inserted[event.EventID]; !exists {
			newEvents = append(newEvents, event)
			inserted[event.EventID] = true
		}
	}
	if len(newEvents) > 0 {
		if err := tx.Create(&newEvents).Error; err != nil {
			return nil, err
		}
	}
	return inserted, nil
}

func updateQueuedEventCounts(tx *gorm.DB, event *model.Event, skipAggregate bool) (*model.Session, error) {
	if !skipAggregate {
		if err := UpsertAggregateCount(tx, event.TenantID, event.EventName, event.Dims); err != nil {
			return nil, err
		}
	}
	var session model.Session
	result := tx.Model(&session).Clauses(clause.Returning{}).Where("id = ?", event.SessionID).
		Updates(map[string]any{"event_count": gorm.Expr("event_count + 1"), "last_seen": time.Now()})
	if result.Error == nil && result.RowsAffected == 0 {
		return nil, nil
	}
	return &session, result.Error
}
