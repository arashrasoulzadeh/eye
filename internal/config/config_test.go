package config

import (
	"os"
	"testing"
)

func TestConfigAddr(t *testing.T) {
	os.Setenv("PORT", "9090")
	defer os.Unsetenv("PORT")

	cfg := &Config{Port: 9090}
	addr := cfg.Addr()
	if addr != ":9090" {
		t.Errorf("Addr() = %q, want :9090", addr)
	}
}