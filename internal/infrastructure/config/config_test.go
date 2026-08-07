package config

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultConfig() *Config {
	return &Config{
		AppName:           "sbterm-server",
		AppVersion:        "dev",
		Port:              ":8080",
		DBMaxConns:        10,
		DBMinConns:        0,
		DBMaxConnLifetime: 30 * time.Minute,
		DBMaxConnIdleTime: 5 * time.Minute,
		RedisURL:          "redis://localhost:6379/0",
		RedisMaxRetries:   3,
		RedisPoolSize:     10,
		RedisMinIdleConns: 0,
		RedisDialTimeout:  5 * time.Second,
		RedisReadTimeout:  3 * time.Second,
		RedisWriteTimeout: 3 * time.Second,
		LogLevel:          "info",
		LogFormat:         "text",
		LogAddSource:      false,
		RateLimitRate:     10,
		RateLimitBurst:    20,
		HTTPReadTimeout:   10 * time.Second,
		HTTPWriteTimeout:  10 * time.Second,
		HTTPIdleTimeout:   60 * time.Second,
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want *Config
	}{
		{
			name: "defaults when env not set",
			want: defaultConfig(),
		},
		{
			name: "all values overridden",
			env: map[string]string{
				"APP_NAME":                  "sbterm",
				"APP_VERSION":               "1.2.3",
				"APP_PORT":                  ":9090",
				"APP_DATABASE_URL":          "postgres://u:p@host:5432/db",
				"APP_DB_MAX_CONNS":          "25",
				"APP_DB_MIN_CONNS":          "2",
				"APP_DB_MAX_CONN_IDLE_TIME": "1m",
				"APP_DB_MAX_CONN_LIFETIME":  "10m",
				"APP_REDIS_URL":             "redis://cache:6379/1",
				"APP_REDIS_MAX_RETRIES":     "5",
				"APP_REDIS_POOL_SIZE":       "20",
				"APP_REDIS_MIN_IDLE_CONNS":  "2",
				"APP_REDIS_DIAL_TIMEOUT":    "2s",
				"APP_REDIS_READ_TIMEOUT":    "1s",
				"APP_REDIS_WRITE_TIMEOUT":   "1500ms",
				"APP_LOG_LEVEL":             "debug",
				"APP_LOG_FORMAT":            "json",
				"APP_LOG_ADD_SOURCE":        "true",
				"APP_RATE_LIMIT_RATE":       "50",
				"APP_RATE_LIMIT_BURST":      "100",
				"APP_HTTP_READ_TIMEOUT":     "5s",
				"APP_HTTP_WRITE_TIMEOUT":    "6s",
				"APP_HTTP_IDLE_TIMEOUT":     "90s",
			},
			want: &Config{
				AppName:           "sbterm",
				AppVersion:        "1.2.3",
				Port:              ":9090",
				DatabaseURL:       "postgres://u:p@host:5432/db",
				DBMaxConns:        25,
				DBMinConns:        2,
				DBMaxConnIdleTime: 1 * time.Minute,
				DBMaxConnLifetime: 10 * time.Minute,
				RedisURL:          "redis://cache:6379/1",
				RedisMaxRetries:   5,
				RedisPoolSize:     20,
				RedisMinIdleConns: 2,
				RedisDialTimeout:  2 * time.Second,
				RedisReadTimeout:  1 * time.Second,
				RedisWriteTimeout: 1500 * time.Millisecond,
				LogLevel:          "debug",
				LogFormat:         "json",
				LogAddSource:      true,
				RateLimitRate:     50,
				RateLimitBurst:    100,
				HTTPReadTimeout:   5 * time.Second,
				HTTPWriteTimeout:  6 * time.Second,
				HTTPIdleTimeout:   90 * time.Second,
			},
		},
		{
			name: "invalid integer falls back to zero",
			env: map[string]string{
				"APP_DB_MAX_CONNS": "not-a-number",
			},
			want: func() *Config {
				cfg := defaultConfig()
				cfg.DBMaxConns = 0
				return cfg
			}(),
		},
		{
			name: "invalid duration falls back to zero",
			env: map[string]string{
				"APP_HTTP_READ_TIMEOUT": "abc",
			},
			want: func() *Config {
				cfg := defaultConfig()
				cfg.HTTPReadTimeout = 0
				return cfg
			}(),
		},
		{
			name: "invalid bool falls back to false",
			env: map[string]string{
				"APP_LOG_ADD_SOURCE": "maybe",
			},
			want: defaultConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := Load()
			require.NoError(t, err)
			assert.Equal(t, *tt.want, *got)
		})
	}
}

func TestLoadFromConfigFile(t *testing.T) {
	tests := []struct {
		name   string
		yaml   string
		env    map[string]string
		mutate func(*Config)
	}{
		{
			name: "config file overrides defaults",
			yaml: "app:\n  name: custom-app\n  version: 2.0.0\nport: \":9090\"\ndb:\n  max_conns: 42\nredis:\n  pool_size: 25\nlog:\n  format: json\n",
			mutate: func(c *Config) {
				c.AppName = "custom-app"
				c.AppVersion = "2.0.0"
				c.Port = ":9090"
				c.DBMaxConns = 42
				c.RedisPoolSize = 25
				c.LogFormat = "json"
			},
		},
		{
			name: "environment overrides config file",
			yaml: "port: \":9090\"\n",
			env:  map[string]string{"APP_PORT": ":7070"},
			mutate: func(c *Config) {
				c.Port = ":7070"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			setDefaults(v)
			v.SetEnvPrefix(EnvPrefix)
			v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
			v.AutomaticEnv()
			v.SetConfigType(ConfigFileType)
			require.NoError(t, v.ReadConfig(bytes.NewBufferString(tt.yaml)))

			for k, val := range tt.env {
				t.Setenv(k, val)
			}

			got := loadFrom(v)
			want := defaultConfig()
			tt.mutate(want)
			assert.Equal(t, *want, *got)
		})
	}
}
