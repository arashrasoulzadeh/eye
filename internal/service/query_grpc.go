package service

import (
	eyev1 "github.com/eye/eye/gen/go/eyev1"
)

// QueryStreamServer implements the Collector gRPC service's QueryStream RPC
// by wrapping QueryService. It embeds UnimplementedCollectorServer so it
// only needs to provide QueryStream — CreateSession/Ingest/IngestStream are
// served by the separate collector binary (server/cmd/collector), which has
// no DB access; this server process owns the DB-backed QueryService instead,
// so QueryStream lives here.
type QueryStreamServer struct {
	eyev1.UnimplementedCollectorServer
	Svc *QueryService
	// PageSize is the default number of events sent per stream message when
	// the request doesn't specify one.
	PageSize int
}

func (s *QueryStreamServer) QueryStream(req *eyev1.QueryStreamRequest, stream eyev1.Collector_QueryStreamServer) error {
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = s.PageSize
	}
	if pageSize <= 0 {
		pageSize = 100
	}

	filter := EventFilter{
		TenantID:     req.GetTenantId(),
		SessionID:    req.GetSessionId(),
		EventName:    req.GetEventName(),
		DimsContains: req.GetDimsContains(),
	}
	if req.GetFromTs() != 0 {
		v := req.GetFromTs()
		filter.FromTS = &v
	}
	if req.GetToTs() != 0 {
		v := req.GetToTs()
		filter.ToTS = &v
	}

	ctx := stream.Context()
	offset := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		events, err := s.Svc.Filter(filter, pageSize, offset)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		pbEvents := make([]*eyev1.Event, 0, len(events))
		for _, e := range events {
			pbEvents = append(pbEvents, &eyev1.Event{
				EventId:       e.EventID,
				EventName:     e.EventName,
				ClientTs:      e.ClientTS,
				Payload:       []byte(e.Payload),
				SessionId:     e.SessionID,
				Source:        e.Source,
				IsAggregated:  e.IsAggregated,
				AggregationId: e.AggregationID,
			})
		}
		if err := stream.Send(&eyev1.QueryStreamResponse{Events: pbEvents}); err != nil {
			return err
		}
		if len(events) < pageSize {
			return nil
		}
		offset += len(events)
	}
}
