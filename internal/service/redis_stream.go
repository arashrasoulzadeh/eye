package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/redis/go-redis/v9"
)

// RedisStream is the durable production implementation of the stream
// interfaces. Messages remain in Redis until acknowledged after the database
// write and aggregate update succeed.
type RedisStream struct {
	client                  *redis.Client
	stream, group, consumer string
}

func NewRedisStream(rawURL, stream, group, consumer string) (*RedisStream, error) {
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}
	return &RedisStream{client: redis.NewClient(options), stream: stream, group: group, consumer: consumer}, nil
}

func (s *RedisStream) Publish(msg StreamMessage) error {
	payload, err := json.Marshal(msg.Event)
	if err != nil {
		return err
	}
	return s.client.XAdd(context.Background(), &redis.XAddArgs{Stream: s.stream, Values: map[string]any{"event": string(payload)}}).Err()
}

func (s *RedisStream) Start(ctx context.Context, handler func(StreamMessage) error) {
	go func() {
		_ = s.client.XGroupCreateMkStream(ctx, s.stream, s.group, "0").Err()
		for ctx.Err() == nil {
			items, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: s.group, Consumer: s.consumer, Streams: []string{s.stream, ">"}, Count: 100, Block: 5 * time.Second}).Result()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			for _, batch := range items {
				for _, item := range batch.Messages {
					var eventPayload string
					if raw, ok := item.Values["event"].(string); ok {
						eventPayload = raw
					}
					var event model.Event
					if json.Unmarshal([]byte(eventPayload), &event) == nil && handler(StreamMessage{Event: &event}) == nil {
						_ = s.client.XAck(ctx, s.stream, s.group, item.ID).Err()
					}
				}
			}
		}
	}()
}
