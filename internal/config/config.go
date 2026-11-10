package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Port           int    `mapstructure:"port"`
	DBPath         string `mapstructure:"db_path"`
	AdminKey       string `mapstructure:"admin_key"`
	APIKeySalt     string `mapstructure:"api_key_salt"`
	RetentionHours int    `mapstructure:"retention_hours"`
	SnapshotDir    string `mapstructure:"snapshot_dir"`
}

func Load() (*Config, error) {
	files := []string{}
	pflag.StringSliceVar(&files, "env", []string{}, "env files to load")
	pflag.Parse()
	for _, f := range files {
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
	viper.SetDefault("api_key_salt", "dev-insecure-salt")
	viper.SetDefault("admin_key", "dev-insecure-admin-key")
	viper.SetDefault("retention_hours", 24*30)
	viper.SetDefault("snapshot_dir", "./snapshots")
	c := &Config{}
	if err := viper.Unmarshal(c); err != nil {
		return nil, err
	}
	return c, nil
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
