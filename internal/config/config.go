package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	CORS     CORSConfig
	Apps     AppsConfig
}

type ServerConfig struct {
	Port int
	Mode string
}

type DatabaseConfig struct {
	Driver      string
	SQLitePath  string `mapstructure:"sqlite_path"`
	PostgresDSN string `mapstructure:"postgres_dsn"`
}

type AuthConfig struct {
	Enabled         bool
	JWTSecret       string `mapstructure:"jwt_secret"`
	PluginSecretKey string `mapstructure:"plugin_secret_key"`
}

type AppsConfig struct {
	PublicBaseURL string `mapstructure:"public_base_url"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8108
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.SQLitePath == "" {
		cfg.Database.SQLitePath = "./data/customerservicecore.db"
	}
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "change-me-in-production-use-long-random-string"
	}
	if cfg.Auth.PluginSecretKey == "" {
		cfg.Auth.PluginSecretKey = cfg.Auth.JWTSecret
	}
	if cfg.Apps.PublicBaseURL == "" {
		cfg.Apps.PublicBaseURL = "http://localhost:5193"
	}
	if len(cfg.CORS.AllowOrigins) == 0 {
		cfg.CORS.AllowOrigins = []string{
			"http://localhost:5193",
			"http://127.0.0.1:5193",
			"http://localhost:5174",
			"http://127.0.0.1:5174",
		}
	}
	return &cfg, nil
}
