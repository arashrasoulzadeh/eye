package main

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"

	eyev1 "github.com/arashrasoulzadeh/eye/gen/go/eyev1"
)

func ctxWithKey(key string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-api-key", key))
}

func TestIngestDedup(t *testing.T) {
	s := newServer()
	ctx := ctxWithKey("tenant-a-key")
	req := &eyev1.IngestRequest{Events: []*eyev1.Event{{EventId: "e1", SessionId: "s1"}}}

	resp, err := s.Ingest(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Accepted != 1 || resp.Rejected != 0 {
		t.Fatalf("expected 1 accepted / 0 rejected, got %+v", resp)
	}

	resp2, err := s.Ingest(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp2.Accepted != 0 || resp2.Rejected != 1 {
		t.Fatalf("expected dup event to be rejected, got %+v", resp2)
	}
}

func TestIngestSessionValidationCrossTenant(t *testing.T) {
	s := newServer()
	ctxA := ctxWithKey("tenant-a-key")
	ctxB := ctxWithKey("tenant-b-key")

	// tenant A creates and owns session s1.
	if _, err := s.Ingest(ctxA, &eyev1.IngestRequest{Events: []*eyev1.Event{{EventId: "e1", SessionId: "s1"}}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// tenant B tries to ingest against tenant A's session.
	resp, err := s.Ingest(ctxB, &eyev1.IngestRequest{Events: []*eyev1.Event{{EventId: "e2", SessionId: "s1"}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Accepted != 0 || resp.Rejected != 1 {
		t.Fatalf("expected cross-tenant session access to be rejected, got %+v", resp)
	}
}

func TestIngestMissingAPIKeyRejected(t *testing.T) {
	s := newServer()
	_, err := s.Ingest(context.Background(), &eyev1.IngestRequest{Events: []*eyev1.Event{{EventId: "e1"}}})
	if err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestIngestRateLimited(t *testing.T) {
	s := newServer()
	ctx := ctxWithKey("tenant-a-key")

	// Drain this tenant's bucket (burst capacity 100) then expect the next
	// call to be rate limited.
	for i := 0; i < 100; i++ {
		ev := &eyev1.Event{EventId: "e" + string(rune('A'+i%26)) + string(rune(i)), SessionId: "s1"}
		if _, err := s.Ingest(ctx, &eyev1.IngestRequest{Events: []*eyev1.Event{ev}}); err != nil {
			t.Fatalf("unexpected error on call %d: %v", i, err)
		}
	}
	if _, err := s.Ingest(ctx, &eyev1.IngestRequest{Events: []*eyev1.Event{{EventId: "overflow", SessionId: "s1"}}}); err == nil {
		t.Fatal("expected rate limit error once bucket capacity is exhausted")
	}
}
