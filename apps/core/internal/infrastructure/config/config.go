package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

const (
	ConfigFileName = "config.core"
	ConfigFileType = "yaml"
	ConfigFilePath = "."
)

// version is overridable at build time via:
//
//	go build -ldflags "-X github.com/nofendian17/sbterm/apps/core/internal/infrastructure/config.version=<tag>"
var version = "dev"

type Config struct {
	App         AppConfig         `mapstructure:"app"`
	Port        string            `mapstructure:"port"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Redis       RedisConfig       `mapstructure:"redis"`
	Log         LogConfig         `mapstructure:"log"`
	RateLimit   RateLimitConfig   `mapstructure:"rate_limit"`
	Auth        AuthConfig        `mapstructure:"auth"`
	HTTP        HTTPConfig        `mapstructure:"http"`
	StockbitAPI StockbitAPIConfig `mapstructure:"stockbit_api"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
}

type RedisConfig struct {
	URL          string        `mapstructure:"url"`
	MaxRetries   int           `mapstructure:"max_retries"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type LogConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	AddSource bool   `mapstructure:"add_source"`
}

type RateLimitConfig struct {
	Rate  int `mapstructure:"rate"`
	Burst int `mapstructure:"burst"`
}

type AuthConfig struct {
	JWTSecret       string        `mapstructure:"jwt_secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_ttl"`
	DefaultUserTTL  time.Duration `mapstructure:"default_user_ttl"`
	BcryptCost      int           `mapstructure:"bcrypt_cost"`
}

type HTTPConfig struct {
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// StockbitAPIConfig points at the apps/api service that proxies the Stockbit
// (Exodus) API (see docs/api.md). apps/core uses it as the upstream source
// for the admin-triggered catalog sync.
type StockbitAPIConfig struct {
	BaseURL string        `mapstructure:"base_url"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// Load reads configuration using viper with the precedence:
// config file > defaults.
func Load() (*Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(ConfigFilePath)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("config: read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "sbterm-core")
	v.SetDefault("app.version", version)
	v.SetDefault("port", ":8082")
	v.SetDefault("database.url", "")
	v.SetDefault("database.max_conns", 10)
	v.SetDefault("database.min_conns", 0)
	v.SetDefault("database.max_conn_lifetime", 30*time.Minute)
	v.SetDefault("database.max_conn_idle_time", 5*time.Minute)
	v.SetDefault("redis.url", "redis://localhost:6379/0")
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 0)
	v.SetDefault("redis.dial_timeout", 5*time.Second)
	v.SetDefault("redis.read_timeout", 3*time.Second)
	v.SetDefault("redis.write_timeout", 3*time.Second)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.add_source", false)
	v.SetDefault("rate_limit.rate", 50)
	v.SetDefault("rate_limit.burst", 100)
	v.SetDefault("auth.jwt_secret", "")
	v.SetDefault("auth.access_ttl", 15*time.Minute)
	v.SetDefault("auth.refresh_ttl", 720*time.Hour)
	v.SetDefault("auth.default_user_ttl", 720*time.Hour)
	v.SetDefault("auth.bcrypt_cost", 12)
	v.SetDefault("http.read_timeout", 10*time.Second)
	v.SetDefault("http.write_timeout", 10*time.Second)
	v.SetDefault("http.idle_timeout", 60*time.Second)
	v.SetDefault("stockbit_api.base_url", "http://localhost:8080")
	v.SetDefault("stockbit_api.timeout", 30*time.Second)
}
