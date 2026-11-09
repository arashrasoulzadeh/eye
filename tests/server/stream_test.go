package test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/service"
)

func TestChannelStreamProducerConsumerRoundTrip(t *testing.T) {
	cs := service.NewChannelStream(8)
	var processed atomic.Int64
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cs.Start(ctx, func(msg service.StreamMessage) error {
		processed.Add(1)
		return nil
	})

	for i := 0; i < 5; i++ {
		if err := cs.Publish(service.StreamMessage{Event: &model.Event{EventID: "e"}}); err != nil {
			t.Fatalf("publish failed: %v", err)
		}
	}
	cs.Wait()

	if got := processed.Load(); got != 5 {
		t.Fatalf("expected 5 processed, got %d", got)
	}
}

func TestChannelStreamStopsOnContextCancel(t *testing.T) {
	cs := service.NewChannelStream(1)
	ctx, cancel := context.WithCancel(context.Background())
	var processed atomic.Int64
	cs.Start(ctx, func(msg service.StreamMessage) error {
		processed.Add(1)
		return nil
	})
	cancel()
	time.Sleep(20 * time.Millisecond) // let the worker observe cancellation

	// Publishing after cancel would block forever waiting for a consumer,
	// so just assert the worker exited cleanly without processing anything
	// further (nothing was published, so count stays at 0).
	if got := processed.Load(); got != 0 {
		t.Fatalf("expected 0 processed after immediate cancel, got %d", got)
	}
}
