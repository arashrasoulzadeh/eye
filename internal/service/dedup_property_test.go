package service

import (
	"testing"
	"testing/quick"
)

func TestEventContentHashProperties(t *testing.T) {
	property := func(tenant, session, name string, timestamp int64, payload, dims []byte) bool {
		first := eventContentHash(tenant, session, name, timestamp, payload, dims)
		second := eventContentHash(tenant, session, name, timestamp, payload, dims)
		return first != "" && first == second
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 1000}); err != nil {
		t.Fatalf("content hash determinism property failed: %v", err)
	}
}

func TestEventContentHashSeparatesIdentityFields(t *testing.T) {
	base := eventContentHash("tenant", "session", "click", 10, []byte(`{"a":1}`), []byte(`{"p":"home"}`))
	cases := []string{
		eventContentHash("other-tenant", "session", "click", 10, []byte(`{"a":1}`), []byte(`{"p":"home"}`)),
		eventContentHash("tenant", "other-session", "click", 10, []byte(`{"a":1}`), []byte(`{"p":"home"}`)),
		eventContentHash("tenant", "session", "purchase", 10, []byte(`{"a":1}`), []byte(`{"p":"home"}`)),
		eventContentHash("tenant", "session", "click", 11, []byte(`{"a":1}`), []byte(`{"p":"home"}`)),
		eventContentHash("tenant", "session", "click", 10, []byte(`{"a":2}`), []byte(`{"p":"home"}`)),
	}
	for i, candidate := range cases {
		if candidate == base {
			t.Fatalf("hash collision for identity variation %d", i)
		}
	}
}
