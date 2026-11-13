package service

import "testing"

func TestEventUserIDPrefersExplicitIdentitySources(t *testing.T) {
	if got := eventUserID(map[string]any{"dims": map[string]any{"userId": "known-1"}}); got != "known-1" { t.Fatalf("got %q", got) }
	if got := eventUserID(map[string]any{"user_id": "known-2", "dims": map[string]any{"userId": "ignored"}}); got != "known-2" { t.Fatalf("got %q", got) }
	if got := eventUserID(map[string]any{"dims": map[string]any{"userId": "  "}}); got != "" { t.Fatalf("got %q", got) }
}
