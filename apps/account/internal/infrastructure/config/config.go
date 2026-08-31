package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

const (
	ConfigFileName = "config.account"
	ConfigFileType = "yaml"
	ConfigFilePath = "."
)

var version = "dev"

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Port      string          `mapstructure:"port"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Log       LogConfig       `mapstructure:"log"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Auth      AuthConfig      `mapstructure:"auth"`
	HTTP      HTTPConfig      `mapstructure:"http"`
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

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(ConfigFilePath)

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		var nf viper.ConfigFileNotFoundError
		if errors.As(err, &nf) {
			return nil, fmt.Errorf("config: file %q not found: %w", ConfigFileName, err)
		}
		return nil, fmt.Errorf("config: read: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}
	// fail fast on missing secret outside dev
	if cfg.Auth.JWTSecret == "" && cfg.App.Version != "dev" {
		return nil, errors.New("config: auth.jwt_secret is required in non-dev builds")
	}
	if cfg.Auth.BcryptCost == 0 {
		cfg.Auth.BcryptCost = 12
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("port", ":8082")
	v.SetDefault("auth.bcrypt_cost", 12)
	v.SetDefault("auth.access_ttl", 15*time.Minute)
	v.SetDefault("auth.refresh_ttl", 720*time.Hour)
	v.SetDefault("auth.default_user_ttl", 720*time.Hour)
}
