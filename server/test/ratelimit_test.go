package test

import (
	"testing"

	"github.com/eye/eye/server/internal/service"
)

func TestRateLimit(t *testing.T) {
	b := service.NewBucket(10, 2)
	if !b.Allow() || !b.Allow() { t.Fatal("first two should pass") }
	if b.Allow() { t.Fatal("third should fail (capacity 2)") }
}
