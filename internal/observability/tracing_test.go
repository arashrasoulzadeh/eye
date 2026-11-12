package observability

import (
	"context"
	"testing"
)

func TestSetupWithoutEndpointIsSafeNoop(t *testing.T) {
	shutdown, err := Setup(context.Background(), "")
	if err != nil { t.Fatal(err) }
	if err := shutdown(context.Background()); err != nil { t.Fatal(err) }
}

func TestSetupRejectsMalformedEndpoint(t *testing.T) {
	if _, err := Setup(context.Background(), "not-an-endpoint"); err == nil { t.Fatal("expected malformed endpoint error") }
}
