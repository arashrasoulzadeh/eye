package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type server struct {
	mu       sync.Mutex
	seen     map[string]struct{}
	sessions map[string]*Session
}

type Session struct {
	ID        string
	TenantID  string
	StartedAt time.Time
	LastSeen  time.Time
	EventCount int64
}

func newServer() *server { return &server{seen: map[string]struct{}{}, sessions: map[string]*Session{}} }

func (s *server) authenticate(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	keys := md.Get("x-api-key")
	if len(keys) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing api key")
	}
	h := sha256.Sum256([]byte(keys[0]))
	return hex.EncodeToString(h[:8]), nil
}

func (s *server) Ingest(ctx context.Context, req *IngestRequest) (*IngestResponse, error) {
	tenant, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	accepted, rejected := 0, 0
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range req.Events {
		key := tenant + ":" + e.EventId
		if _, dup := s.seen[key]; dup {
			rejected++
			continue
		}
		s.seen[key] = struct{}{}
		// session tracking
		if e.SessionId != "" {
			sess, ok := s.sessions[e.SessionId]
			if !ok {
				sess = &Session{ID: e.SessionId, TenantID: tenant, StartedAt: time.Now()}
				s.sessions[e.SessionId] = sess
			}
			sess.LastSeen = time.Now()
			sess.EventCount++
		}
		accepted++
	}
	return &IngestResponse{Accepted: int32(accepted), Rejected: int32(rejected)}, nil
}

func (s *server) IngestStream(stream interface{ Recv() (*IngestRequest, error); SendAndClose(*IngestResponse) error }) error {
	// placeholder for streaming ingest
	return fmt.Errorf("not implemented")
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	gs := grpc.NewServer()
	// register would require generated code; keep scaffold
	log.Println("collector listening on :50051")
	if err := gs.Serve(lis); err != nil {
		log.Fatal(err)
	}
}