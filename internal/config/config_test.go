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

func TestValidateForProductionRejectsMissingAdminKey(t *testing.T) {
	os.Setenv("EYE_ENV", "production")
	defer os.Unsetenv("EYE_ENV")

	cfg := &Config{APIKeySalt: "salt-from-secret"}
	if err := cfg.ValidateForProduction(); err == nil {
		t.Fatal("expected missing admin key to be rejected")
	}
}

func TestValidateForProductionRejectsMissingAPISalt(t *testing.T) {
	os.Setenv("EYE_ENV", "production")
	defer os.Unsetenv("EYE_ENV")

	cfg := &Config{AdminKey: "admin-from-secret"}
	if err := cfg.ValidateForProduction(); err == nil {
		t.Fatal("expected missing API key salt to be rejected")
	}
}

func TestValidateForProductionRejectsDevelopmentAdminWithValidSalt(t *testing.T) {
	os.Setenv("EYE_ENV", "production")
	defer os.Unsetenv("EYE_ENV")

	cfg := &Config{AdminKey: "dev-insecure-admin-key", APIKeySalt: "salt-from-secret"}
	if err := cfg.ValidateForProduction(); err == nil {
		t.Fatal("expected development admin key to be rejected")
	}
}

func TestValidateForProductionRejectsDevelopmentSaltWithValidAdmin(t *testing.T) {
	os.Setenv("EYE_ENV", "production")
	defer os.Unsetenv("EYE_ENV")

	cfg := &Config{AdminKey: "admin-from-secret", APIKeySalt: "dev-insecure-salt"}
	if err := cfg.ValidateForProduction(); err == nil {
		t.Fatal("expected development API key salt to be rejected")
	}
}

func TestValidateForProductionAllowsDevelopmentCredentialsOutsideProduction(t *testing.T) {
	os.Setenv("EYE_ENV", "development")
	defer os.Unsetenv("EYE_ENV")

	cfg := &Config{AdminKey: "dev-insecure-admin-key", APIKeySalt: "dev-insecure-salt"}
	if err := cfg.ValidateForProduction(); err != nil {
		t.Fatalf("development credentials should be allowed outside production: %v", err)
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
