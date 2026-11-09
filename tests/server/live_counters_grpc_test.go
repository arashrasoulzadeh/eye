package test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	eyev1 "github.com/arashrasoulzadeh/eye/gen/go/eyev1"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func startLiveCountersServer(t *testing.T, hub *service.LiveCounterHub) (eyev1.CollectorClient, func()) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	eyev1.RegisterCollectorServer(gs, &service.LiveCountersServer{Hub: hub})
	go func() { _ = gs.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	client := eyev1.NewCollectorClient(conn)
	return client, func() {
		conn.Close()
		gs.Stop()
	}
}

func TestLiveCountersSendsImmediateSnapshotThenLiveUpdates(t *testing.T) {
	hub := service.NewLiveCounterHub()
	hub.Increment("tenant-a") // pre-existing count before the stream opens

	client, cleanup := startLiveCountersServer(t, hub)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := client.LiveCounters(ctx, &eyev1.LiveCountersRequest{TenantId: "tenant-a"})
	if err != nil {
		t.Fatalf("LiveCounters: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("expected immediate snapshot, got err: %v", err)
	}
	if resp.GetCount() != 1 {
		t.Fatalf("expected initial snapshot count 1, got %d", resp.GetCount())
	}

	hub.Increment("tenant-a")
	resp, err = stream.Recv()
	if err != nil {
		t.Fatalf("expected live update, got err: %v", err)
	}
	if resp.GetCount() != 2 {
		t.Fatalf("expected live update count 2, got %d", resp.GetCount())
	}
}

func TestLiveCountersIgnoresOtherTenantsUpdates(t *testing.T) {
	hub := service.NewLiveCounterHub()
	client, cleanup := startLiveCountersServer(t, hub)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := client.LiveCounters(ctx, &eyev1.LiveCountersRequest{TenantId: "tenant-a"})
	if err != nil {
		t.Fatalf("LiveCounters: %v", err)
	}
	// initial snapshot (0)
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("expected initial snapshot, got err: %v", err)
	}

	hub.Increment("tenant-b")

	done := make(chan struct{})
	go func() {
		stream.Recv()
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("received an update for a different tenant")
	case <-time.After(100 * time.Millisecond):
	}
}
