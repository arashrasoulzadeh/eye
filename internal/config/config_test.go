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

func TestValidateForProductionRejectsDevelopmentCredentials(t *testing.T) {
	os.Setenv("EYE_ENV", "production")
	defer os.Unsetenv("EYE_ENV")

	cfg := &Config{AdminKey: "dev-insecure-admin-key", APIKeySalt: "dev-insecure-salt"}
	if err := cfg.ValidateForProduction(); err == nil {
		t.Fatal("expected production validation to reject development credentials")
	}
}

func TestValidateForProductionAcceptsConfiguredCredentials(t *testing.T) {
	os.Setenv("EYE_ENV", "production")
	defer os.Unsetenv("EYE_ENV")

	cfg := &Config{AdminKey: "admin-from-secret", APIKeySalt: "salt-from-secret"}
	if err := cfg.ValidateForProduction(); err != nil {
		t.Fatalf("expected configured production credentials to pass: %v", err)
	}
}
