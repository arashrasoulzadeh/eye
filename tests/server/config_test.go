package test

import (
	"os"
	"testing"

	"github.com/eye/eye/internal/config"
)

func TestConfigLoadDefaultsAndEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("API_KEY_SALT", "test-salt")
	defer os.Unsetenv("PORT")
	defer os.Unsetenv("API_KEY_SALT")

	c, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if c.Port != 9090 {
		t.Fatalf("expected port 9090 from env, got %d", c.Port)
	}
	if c.APIKeySalt != "test-salt" {
		t.Fatalf("expected api key salt from env, got %q", c.APIKeySalt)
	}
	if c.DBPath == "" {
		t.Fatalf("expected default db path to be set")
	}
}
