package test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	eyev1 "github.com/arashrasoulzadeh/eye/gen/go/eyev1"
	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func newQueryStreamTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// A plain ":memory:" DSN gives each pooled connection its own empty DB;
	// pin to a single connection so every query sees the same data.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.Event{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// startQueryStreamServer boots a real gRPC server over an in-memory
// (bufconn) listener running QueryStreamServer, and returns a connected
// client plus a cleanup func.
func startQueryStreamServer(t *testing.T, qs *service.QueryService, pageSize int) (eyev1.CollectorClient, func()) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	eyev1.RegisterCollectorServer(gs, &service.QueryStreamServer{Svc: qs, PageSize: pageSize})
	go func() { _ = gs.Serve(lis) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_ = ctx
	client := eyev1.NewCollectorClient(conn)
	return client, func() {
		conn.Close()
		gs.Stop()
	}
}

func seedQueryStreamEvents(t *testing.T, db *gorm.DB, n int, tenant, name string) {
	t.Helper()
	for i := 0; i < n; i++ {
		e := model.Event{
			EventID:   name + "-" + tenant + "-" + time.Now().Add(time.Duration(i)*time.Millisecond).String(),
			SessionID: "s1",
			TenantID:  tenant,
			EventName: name,
			ClientTS:  int64(i),
			Payload:   "{}",
			Dims:      "{}",
			Source:    "client",
		}
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}
}

func TestQueryStreamReturnsAllMatchingEventsAcrossMultiplePages(t *testing.T) {
	db := newQueryStreamTestDB(t)
	seedQueryStreamEvents(t, db, 250, "tenant-a", "click")
	seedQueryStreamEvents(t, db, 10, "tenant-b", "click") // different tenant, must be excluded

	qs := &service.QueryService{DB: db}
	client, cleanup := startQueryStreamServer(t, qs, 100) // forces 3 pages for 250 events
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := client.QueryStream(ctx, &eyev1.QueryStreamRequest{TenantId: "tenant-a", EventName: "click"})
	if err != nil {
		t.Fatalf("QueryStream: %v", err)
	}

	total := 0
	pages := 0
	for {
		resp, err := stream.Recv()
		if err != nil {
			break // EOF ends the stream
		}
		pages++
		total += len(resp.GetEvents())
		for _, e := range resp.GetEvents() {
			if e.GetEventName() != "click" {
				t.Fatalf("unexpected event in stream: %+v", e)
			}
		}
	}
	if total != 250 {
		t.Fatalf("expected 250 streamed events, got %d", total)
	}
	if pages < 3 {
		t.Fatalf("expected at least 3 pages for 250 events at page size 100, got %d", pages)
	}
}

func TestQueryStreamRespectsFilters(t *testing.T) {
	db := newQueryStreamTestDB(t)
	seedQueryStreamEvents(t, db, 5, "tenant-a", "click")
	seedQueryStreamEvents(t, db, 5, "tenant-a", "page_view")

	qs := &service.QueryService{DB: db}
	client, cleanup := startQueryStreamServer(t, qs, 100)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := client.QueryStream(ctx, &eyev1.QueryStreamRequest{TenantId: "tenant-a", EventName: "page_view"})
	if err != nil {
		t.Fatalf("QueryStream: %v", err)
	}
	total := 0
	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		total += len(resp.GetEvents())
	}
	if total != 5 {
		t.Fatalf("expected 5 page_view events, got %d", total)
	}
}

func TestQueryStreamEmptyResultClosesCleanly(t *testing.T) {
	db := newQueryStreamTestDB(t)
	qs := &service.QueryService{DB: db}
	client, cleanup := startQueryStreamServer(t, qs, 100)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := client.QueryStream(ctx, &eyev1.QueryStreamRequest{TenantId: "nope"})
	if err != nil {
		t.Fatalf("QueryStream: %v", err)
	}
	if _, err := stream.Recv(); err == nil {
		t.Fatalf("expected immediate stream close for no matching events")
	}
}
