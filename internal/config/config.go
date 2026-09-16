package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	CORS     CORSConfig
	Apps     AppsConfig
	LLM      LLMConfig
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

type LLMConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Provider   string `mapstructure:"provider"`
	APIBase    string `mapstructure:"api_base"`
	APIKey     string `mapstructure:"api_key"`
	Model      string `mapstructure:"model"`
	TimeoutSec int    `mapstructure:"timeout_sec"`
	MaxChars   int    `mapstructure:"max_chars"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault("llm.enabled", true)
	v.SetDefault("llm.api_base", "https://api.deepseek.com")
	v.SetDefault("llm.model", "deepseek-chat")
	v.SetDefault("llm.timeout_sec", 8)
	v.SetDefault("llm.max_chars", 40)
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
	if cfg.LLM.APIKey == "" {
		cfg.LLM.APIKey = os.Getenv("LLM_API_KEY")
	}
	if cfg.LLM.APIKey == "" {
		cfg.LLM.APIKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	if cfg.LLM.APIBase == "" {
		cfg.LLM.APIBase = "https://api.deepseek.com"
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "deepseek-chat"
	}
	if cfg.LLM.TimeoutSec <= 0 {
		cfg.LLM.TimeoutSec = 8
	}
	if cfg.LLM.MaxChars <= 0 {
		cfg.LLM.MaxChars = 40
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
