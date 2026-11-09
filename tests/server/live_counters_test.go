package test

import (
	"testing"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/service"
)

func TestLiveCounterHubIncrementUpdatesSnapshot(t *testing.T) {
	h := service.NewLiveCounterHub()
	h.Increment("tenant-a")
	h.Increment("tenant-a")
	h.Increment("tenant-b")

	if got := h.Snapshot("tenant-a"); got != 2 {
		t.Fatalf("expected tenant-a count 2, got %d", got)
	}
	if got := h.Snapshot("tenant-b"); got != 1 {
		t.Fatalf("expected tenant-b count 1, got %d", got)
	}
	if got := h.Snapshot("tenant-c"); got != 0 {
		t.Fatalf("expected unseen tenant count 0, got %d", got)
	}
}

func TestLiveCounterHubSubscribeReceivesUpdatesForItsTenantOnly(t *testing.T) {
	h := service.NewLiveCounterHub()
	ch, unsubscribe := h.Subscribe("tenant-a")
	defer unsubscribe()

	h.Increment("tenant-b") // different tenant, must not be delivered
	h.Increment("tenant-a")

	select {
	case count := <-ch:
		if count != 1 {
			t.Fatalf("expected delivered count 1, got %d", count)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for update")
	}

	select {
	case count := <-ch:
		t.Fatalf("unexpected second update (tenant-b increment leaked): %d", count)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestLiveCounterHubUnsubscribeStopsDelivery(t *testing.T) {
	h := service.NewLiveCounterHub()
	ch, unsubscribe := h.Subscribe("tenant-a")
	unsubscribe()

	h.Increment("tenant-a")

	select {
	case <-ch:
		t.Fatal("received update after unsubscribe")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestLiveCounterHubMultipleSubscribersEachReceiveUpdates(t *testing.T) {
	h := service.NewLiveCounterHub()
	ch1, unsub1 := h.Subscribe("tenant-a")
	defer unsub1()
	ch2, unsub2 := h.Subscribe("tenant-a")
	defer unsub2()

	h.Increment("tenant-a")

	for _, ch := range []chan int64{ch1, ch2} {
		select {
		case count := <-ch:
			if count != 1 {
				t.Fatalf("expected count 1, got %d", count)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for update")
		}
	}
}
