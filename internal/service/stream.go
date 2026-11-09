package service

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// StreamMessage is a unit of work flowing through the ingest pipeline: one
// event to persist and fold into materialized aggregates.
type StreamMessage struct {
	Event *model.Event
}

// StreamProducer publishes a message onto the stream. A production
// deployment backed by Kafka/Redpanda would implement this against a real
// network broker; ChannelStream below is the in-process substitute used in
// dev.
type StreamProducer interface {
	Publish(msg StreamMessage) error
}

// StreamConsumer starts a background worker that calls handler for every
// published message until ctx is cancelled.
type StreamConsumer interface {
	Start(ctx context.Context, handler func(StreamMessage) error)
}

// ChannelStream is an in-process, in-memory producer+consumer backed by a
// buffered Go channel. CLAUDE.md's dev overrides rule out Docker/external
// brokers in dev (SQLite only, no infra), so this fulfills the same
// architectural role as a Kafka/Redpanda topic — decoupling the ingest
// request path from persistence/aggregation via a real concurrent queue, a
// goroutine worker, and bounded-channel backpressure — but it is honestly
// NOT Kafka/Redpanda: no network broker, no partitioning, no durability
// across process restarts. Swap it for a real implementation of
// StreamProducer/StreamConsumer against an actual broker in production.
type ChannelStream struct {
	ch chan StreamMessage
	wg sync.WaitGroup
}

func NewChannelStream(buffer int) *ChannelStream {
	return &ChannelStream{ch: make(chan StreamMessage, buffer)}
}

func (c *ChannelStream) Publish(msg StreamMessage) error {
	c.wg.Add(1)
	c.ch <- msg
	return nil
}

func (c *ChannelStream) Start(ctx context.Context, handler func(StreamMessage) error) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-c.ch:
				_ = handler(msg)
				c.wg.Done()
			}
		}
	}()
}

// Wait blocks until every published message has been handled. Used by tests
// (and anything that wants an at-least-processed guarantee) to synchronize
// with the async consumer without polling.
func (c *ChannelStream) Wait() { c.wg.Wait() }

// QueueDepth reports how many messages are currently buffered and not yet
// picked up by the consumer goroutine — the Phase 9 "queue depth" metric's
// source of truth for this in-process stream.
func (c *ChannelStream) QueueDepth() int { return len(c.ch) }

// UpsertAggregateCount folds one event occurrence into the materialized
// per-(tenant, event_name, dims) count — the "materialized aggregates" piece
// of Phase 7. dimsKey is the event's raw Dims JSON string, used verbatim as
// the grouping key.
func UpsertAggregateCount(db *gorm.DB, tenantID, eventName, dimsKey string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var ac model.AggregateCount
		err := tx.Where("tenant_id = ? AND event_name = ? AND dims_key = ?", tenantID, eventName, dimsKey).First(&ac).Error
		if err == gorm.ErrRecordNotFound {
			ac = model.AggregateCount{TenantID: tenantID, EventName: eventName, DimsKey: dimsKey, Count: 1, UpdatedAt: time.Now().Unix()}
			return tx.Create(&ac).Error
		}
		if err != nil {
			return err
		}
		// Not tx.Save(&ac): AggregateCount's composite primary key can
		// legitimately have an empty-string DimsKey (the "no dims" case), and
		// GORM's Save() treats any zero-value primary key field as "this is a
		// new record," silently turning the intended UPDATE into a duplicate
		// INSERT that fails the table's unique constraint. An explicit
		// Updates() against the matched WHERE clause has no such ambiguity.
		return tx.Model(&model.AggregateCount{}).
			Where("tenant_id = ? AND event_name = ? AND dims_key = ?", tenantID, eventName, dimsKey).
			Updates(map[string]any{"count": ac.Count + 1, "updated_at": time.Now().Unix()}).Error
	})
}

// GetAggregateCount reads back a materialized count. Returns 0, nil if no
// events matching (tenantID, eventName, dimsKey) have been consumed yet.
func GetAggregateCount(db *gorm.DB, tenantID, eventName, dimsKey string) (int64, error) {
	var ac model.AggregateCount
	err := db.Where("tenant_id = ? AND event_name = ? AND dims_key = ?", tenantID, eventName, dimsKey).First(&ac).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return ac.Count, nil
}
