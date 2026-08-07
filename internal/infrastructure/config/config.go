package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	EnvPrefix      = "APP"
	ConfigFileName = "config"
	ConfigFileType = "yaml"
	ConfigFilePath = "."
)

// version is overridable at build time via:
//
//	go build -ldflags "-X github.com/nofendian17/sbterm-server/internal/infrastructure/config.version=<tag>"
var version = "dev"

type Config struct {
	AppName    string
	AppVersion string

	Port string

	DatabaseURL       string
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration

	RedisURL          string
	RedisMaxRetries   int
	RedisPoolSize     int
	RedisMinIdleConns int
	RedisDialTimeout  time.Duration
	RedisReadTimeout  time.Duration
	RedisWriteTimeout time.Duration

	LogLevel     string
	LogFormat    string
	LogAddSource bool

	RateLimitRate  int
	RateLimitBurst int

	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
}

// Load reads configuration using viper with the precedence:
// environment variables (APP_ prefix) > config file > defaults.
func Load() (*Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.BindEnv("app.name", "APP_NAME")
	v.BindEnv("app.version", "APP_VERSION")

	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(ConfigFilePath)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("config: read config file: %w", err)
		}
	}

	return loadFrom(v), nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "sbterm-server")
	v.SetDefault("app.version", version)
	v.SetDefault("port", ":8080")
	v.SetDefault("database.url", "")
	v.SetDefault("db.max_conns", 10)
	v.SetDefault("db.min_conns", 0)
	v.SetDefault("db.max_conn_lifetime", 30*time.Minute)
	v.SetDefault("db.max_conn_idle_time", 5*time.Minute)
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
	v.SetDefault("rate_limit.rate", 10)
	v.SetDefault("rate_limit.burst", 20)
	v.SetDefault("http.read_timeout", 10*time.Second)
	v.SetDefault("http.write_timeout", 10*time.Second)
	v.SetDefault("http.idle_timeout", 60*time.Second)
}

func loadFrom(v *viper.Viper) *Config {
	return &Config{
		AppName:           v.GetString("app.name"),
		AppVersion:        v.GetString("app.version"),
		Port:              v.GetString("port"),
		DatabaseURL:       v.GetString("database.url"),
		DBMaxConns:        v.GetInt32("db.max_conns"),
		DBMinConns:        v.GetInt32("db.min_conns"),
		DBMaxConnLifetime: v.GetDuration("db.max_conn_lifetime"),
		DBMaxConnIdleTime: v.GetDuration("db.max_conn_idle_time"),
		RedisURL:          v.GetString("redis.url"),
		RedisMaxRetries:   v.GetInt("redis.max_retries"),
		RedisPoolSize:     v.GetInt("redis.pool_size"),
		RedisMinIdleConns: v.GetInt("redis.min_idle_conns"),
		RedisDialTimeout:  v.GetDuration("redis.dial_timeout"),
		RedisReadTimeout:  v.GetDuration("redis.read_timeout"),
		RedisWriteTimeout: v.GetDuration("redis.write_timeout"),
		LogLevel:          v.GetString("log.level"),
		LogFormat:         v.GetString("log.format"),
		LogAddSource:      v.GetBool("log.add_source"),
		RateLimitRate:     v.GetInt("rate_limit.rate"),
		RateLimitBurst:    v.GetInt("rate_limit.burst"),
		HTTPReadTimeout:   v.GetDuration("http.read_timeout"),
		HTTPWriteTimeout:  v.GetDuration("http.write_timeout"),
		HTTPIdleTimeout:   v.GetDuration("http.idle_timeout"),
	}
}
