package main

type Event struct {
	EventId   string
	EventName string
	ClientTs  int64
	Payload   []byte
	Dims      map[string]string
	SessionId string
}

type IngestRequest struct {
	TenantId string
	Events   []*Event
}

type IngestResponse struct {
	Accepted int32
	Rejected int32
}