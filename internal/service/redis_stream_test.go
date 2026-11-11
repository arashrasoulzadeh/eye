package service

import "testing"

func TestNewRedisStreamRejectsInvalidURL(t *testing.T) {
	if _, err := NewRedisStream("not-a-redis-url", "events", "consumers", "server"); err == nil {
		t.Fatal("expected invalid Redis URL to be rejected")
	}
}

func TestNewRedisStreamPreservesRuntimeNames(t *testing.T) {
	stream, err := NewRedisStream("redis://localhost:6379/0", "eye-events", "eye-consumers", "eye-server")
	if err != nil {
		t.Fatal(err)
	}
	if stream.stream != "eye-events" || stream.group != "eye-consumers" || stream.consumer != "eye-server" {
		t.Fatalf("unexpected Redis stream configuration: %#v", stream)
	}
	_ = stream.client.Close()
}
