package service

import (
	eyev1 "github.com/arashrasoulzadeh/eye/gen/go/eyev1"
)

// LiveCountersServer implements the Collector gRPC service's LiveCounters
// RPC by wrapping LiveCounterHub. Like QueryStreamServer, it embeds
// UnimplementedCollectorServer so it only needs to provide this one method.
type LiveCountersServer struct {
	eyev1.UnimplementedCollectorServer
	Hub *LiveCounterHub
}

func (s *LiveCountersServer) LiveCounters(req *eyev1.LiveCountersRequest, stream eyev1.Collector_LiveCountersServer) error {
	return liveCounters(s.Hub, req, stream)
}

// liveCounters is the shared implementation used by both LiveCountersServer
// (standalone, for unit tests) and QueryStreamServer (the combined type
// actually registered by server/cmd/server/main.go, since only one
// CollectorServer implementation can be registered per gRPC server).
func liveCounters(hub *LiveCounterHub, req *eyev1.LiveCountersRequest, stream eyev1.Collector_LiveCountersServer) error {
	tenantID := req.GetTenantId()

	// Subscribe before sending the snapshot so no Increment between the
	// snapshot read and the subscription call is ever missed.
	ch, unsubscribe := hub.Subscribe(tenantID)
	defer unsubscribe()

	if err := stream.Send(&eyev1.LiveCountersResponse{Count: hub.Snapshot(tenantID)}); err != nil {
		return err
	}

	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case count := <-ch:
			if err := stream.Send(&eyev1.LiveCountersResponse{Count: count}); err != nil {
				return err
			}
		}
	}
}
