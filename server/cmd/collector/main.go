package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	eyev1 "github.com/arashrasoulzadeh/eye/gen/go/eyev1"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

type server struct {
	eyev1.UnimplementedCollectorServer

	mu       sync.Mutex
	seen     map[string]struct{}
	sessions map[string]*Session
	limiters map[string]*service.Bucket
}

type Session struct {
	ID         string
	TenantID   string
	StartedAt  time.Time
	LastSeen   time.Time
	EventCount int64
}

func newServer() *server {
	return &server{
		seen:     map[string]struct{}{},
		sessions: map[string]*Session{},
		limiters: map[string]*service.Bucket{},
	}
}

func (s *server) authenticate(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	keys := md.Get("x-api-key")
	if len(keys) == 0 || keys[0] == "" {
		return "", status.Error(codes.Unauthenticated, "missing api key")
	}
	h := sha256.Sum256([]byte(keys[0]))
	return hex.EncodeToString(h[:8]), nil
}

// bucketFor returns (creating if needed) the per-tenant token bucket used to
// rate-limit ingest calls.
func (s *server) bucketFor(tenant string) *service.Bucket {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.limiters[tenant]
	if !ok {
		b = service.NewBucket(50, 100) // 50 events/sec sustained, burst of 100
		s.limiters[tenant] = b
	}
	return b
}

func (s *server) CreateSession(ctx context.Context, req *eyev1.CreateSessionRequest) (*eyev1.CreateSessionResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := hex.EncodeToString([]byte(req.GetTenantId() + ":" + time.Now().String()))[:32]
	sess := &Session{ID: id, TenantID: req.GetTenantId(), StartedAt: time.Now(), LastSeen: time.Now()}
	s.sessions[id] = sess
	expires := time.Now().Add(24 * time.Hour).Unix()
	return &eyev1.CreateSessionResponse{SessionId: id, ExpiresAt: expires}, nil
}

func (s *server) Ingest(ctx context.Context, req *eyev1.IngestRequest) (*eyev1.IngestResponse, error) {
	tenant, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if !s.bucketFor(tenant).Allow() {
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}

	accepted, rejected := 0, 0
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range req.GetEvents() {
		sessionID := e.GetSessionId()

		// Session validation: the event's session must belong to this tenant.
		sess, ok := s.sessions[sessionID]
		if sessionID != "" && ok && sess.TenantID != tenant {
			rejected++
			continue
		}

		// Idempotency: dedup on (session_id, event_id).
		key := tenant + ":" + sessionID + ":" + e.GetEventId()
		if _, dup := s.seen[key]; dup {
			rejected++
			continue
		}
		s.seen[key] = struct{}{}

		if sessionID != "" {
			if !ok {
				sess = &Session{ID: sessionID, TenantID: tenant, StartedAt: time.Now()}
				s.sessions[sessionID] = sess
			}
			sess.LastSeen = time.Now()
			sess.EventCount++
		}
		accepted++
	}
	return &eyev1.IngestResponse{Accepted: int32(accepted), Rejected: int32(rejected)}, nil
}

func (s *server) IngestStream(stream grpc.ClientStreamingServer[eyev1.IngestRequest, eyev1.IngestResponse]) error {
	totalAccepted, totalRejected := int32(0), int32(0)
	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}
		resp, err := s.Ingest(stream.Context(), req)
		if err != nil {
			return err
		}
		totalAccepted += resp.GetAccepted()
		totalRejected += resp.GetRejected()
	}
	return stream.SendAndClose(&eyev1.IngestResponse{Accepted: totalAccepted, Rejected: totalRejected})
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Error("listen failed", "err", err)
		os.Exit(1)
	}
	gs := grpc.NewServer()
	eyev1.RegisterCollectorServer(gs, newServer())

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("eye.v1.Collector", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(gs, healthSrv)

	reflection.Register(gs)

	logger.Info("collector listening", "addr", ":50051")
	if err := gs.Serve(lis); err != nil {
		logger.Error("serve failed", "err", err)
		os.Exit(1)
	}
}
