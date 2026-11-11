package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Port                int    `mapstructure:"port"`
	DBPath              string `mapstructure:"db_path"`
	DBDriver            string `mapstructure:"db_driver"`
	DatabaseURL         string `mapstructure:"database_url"`
	CORSOrigins         string `mapstructure:"cors_origins"`
	APIKeyTTLHours      int    `mapstructure:"api_key_ttl_hours"`
	BrokerURL           string `mapstructure:"broker_url"`
	BrokerStream        string `mapstructure:"broker_stream"`
	BrokerGroup         string `mapstructure:"broker_group"`
	BrokerConsumer      string `mapstructure:"broker_consumer"`
	EnablePostgresRLS   bool   `mapstructure:"enable_postgres_rls"`
	AdminKey            string `mapstructure:"admin_key"`
	APIKeySalt          string `mapstructure:"api_key_salt"`
	RetentionHours      int    `mapstructure:"retention_hours"`
	SnapshotDir         string `mapstructure:"snapshot_dir"`
	PIIFields           string `mapstructure:"pii_fields"`
	SchemaMode          string `mapstructure:"schema_mode"`
	DeadLetterNotifyURL string `mapstructure:"dead_letter_notify_url"`
}

var envFiles []string

func Load() (*Config, error) {
	if pflag.Lookup("env") == nil {
		pflag.StringSliceVar(&envFiles, "env", []string{}, "env files to load")
	}
	pflag.Parse()
	for _, f := range envFiles {
		viper.SetConfigFile(f)
		viper.SetConfigType("yaml")
		_ = viper.MergeInConfig()
		// try env file
		viper.SetConfigFile(f)
		viper.SetConfigType("env")
		_ = viper.MergeInConfig()
	}
	viper.AutomaticEnv()
	viper.SetDefault("port", 8080)
	viper.SetDefault("db_path", "eye.db")
	viper.SetDefault("db_driver", "sqlite")
	viper.SetDefault("api_key_salt", "dev-insecure-salt")
	viper.SetDefault("admin_key", "dev-insecure-admin-key")
	viper.SetDefault("retention_hours", 24*30)
	viper.SetDefault("snapshot_dir", "./snapshots")
	viper.SetDefault("broker_stream", "eye-events")
	viper.SetDefault("broker_group", "eye-consumers")
	viper.SetDefault("broker_consumer", "eye-server")
	c := &Config{}
	if err := viper.Unmarshal(c); err != nil {
		return nil, err
	}
	if value, ok := secretFromFile("ADMIN_KEY", c.AdminKey); ok {
		c.AdminKey = value
	}
	if value, ok := secretFromFile("API_KEY_SALT", c.APIKeySalt); ok {
		c.APIKeySalt = value
	}
	if value, ok := secretFromFile("DATABASE_URL", c.DatabaseURL); ok {
		c.DatabaseURL = value
	}
	return c, nil
}

func secretFromFile(name, current string) (string, bool) {
	path := strings.TrimSpace(os.Getenv(name + "_FILE"))
	if path == "" {
		return current, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return current, false
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return current, false
	}
	return value, true
}

func (c *Config) Addr() string { return fmt.Sprintf(":%d", c.Port) }

// ValidateForProduction rejects credentials that are safe only for local
// development. Callers can keep the convenient development defaults while
// making production startup fail closed when EYE_ENV=production.
func (c *Config) ValidateForProduction() error {
	if strings.EqualFold(os.Getenv("EYE_ENV"), "production") {
		if c.AdminKey == "" || c.AdminKey == "dev-insecure-admin-key" {
			return fmt.Errorf("admin_key must be explicitly configured in production")
		}
		if c.APIKeySalt == "" || c.APIKeySalt == "dev-insecure-salt" {
			return fmt.Errorf("api_key_salt must be explicitly configured in production")
		}
	}
	return nil
}
