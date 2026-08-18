package config

import (
	"bytes"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:    "sbterm-server",
			Version: "dev",
		},
		Port: ":8080",
		Database: DatabaseConfig{
			MaxConns:        10,
			MinConns:        0,
			MaxConnLifetime: 30 * time.Minute,
			MaxConnIdleTime: 5 * time.Minute,
		},
		Redis: RedisConfig{
			URL:          "redis://localhost:6379/0",
			MaxRetries:   3,
			PoolSize:     10,
			MinIdleConns: 0,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		Stockbit: StockbitConfig{
			BaseURL:    "https://exodus.stockbit.com",
			Timeout:    30 * time.Second,
			RetryCount: 3,
		},
		Log: LogConfig{
			Level:     "info",
			Format:    "text",
			AddSource: false,
		},
		RateLimit: RateLimitConfig{
			Rate:  10,
			Burst: 20,
		},
		HTTP: HTTPConfig{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func TestLoad(t *testing.T) {
	got, err := Load()
	require.NoError(t, err)
	assert.Equal(t, *defaultConfig(), *got)
}

func TestLoadFromConfigFile(t *testing.T) {
	tests := []struct {
		name   string
		yaml   string
		mutate func(*Config)
	}{
		{
			name: "config file overrides defaults",
			yaml: `app:
  name: custom-app
  version: 2.0.0
port: ":9090"
database:
  max_conns: 42
redis:
  pool_size: 25
log:
  format: json
`,
			mutate: func(c *Config) {
				c.App.Name = "custom-app"
				c.App.Version = "2.0.0"
				c.Port = ":9090"
				c.Database.MaxConns = 42
				c.Redis.PoolSize = 25
				c.Log.Format = "json"
			},
		},
		{
			name: "all values overridden",
			yaml: `app:
  name: sbterm
  version: 1.2.3
port: ":9090"
database:
  url: postgres://u:p@host:5432/db
  max_conns: 25
  min_conns: 2
  max_conn_idle_time: 1m
  max_conn_lifetime: 10m
redis:
  url: redis://cache:6379/1
  max_retries: 5
  pool_size: 20
  min_idle_conns: 2
  dial_timeout: 2s
  read_timeout: 1s
  write_timeout: 1500ms
stockbit:
  base_url: https://exodus.example.com
  timeout: 15s
  retry_count: 1
  player_id: p123
  username: budi
  password: secret
log:
  level: debug
  format: json
  add_source: true
rate_limit:
  rate: 50
  burst: 100
http:
  read_timeout: 5s
  write_timeout: 6s
  idle_timeout: 90s
`,
			mutate: func(c *Config) {
				c.App.Name = "sbterm"
				c.App.Version = "1.2.3"
				c.Port = ":9090"
				c.Database = DatabaseConfig{
					URL:             "postgres://u:p@host:5432/db",
					MaxConns:        25,
					MinConns:        2,
					MaxConnIdleTime: 1 * time.Minute,
					MaxConnLifetime: 10 * time.Minute,
				}
				c.Redis = RedisConfig{
					URL:          "redis://cache:6379/1",
					MaxRetries:   5,
					PoolSize:     20,
					MinIdleConns: 2,
					DialTimeout:  2 * time.Second,
					ReadTimeout:  1 * time.Second,
					WriteTimeout: 1500 * time.Millisecond,
				}
				c.Stockbit = StockbitConfig{
					BaseURL:    "https://exodus.example.com",
					Timeout:    15 * time.Second,
					RetryCount: 1,
					PlayerID:   "p123",
					Username:   "budi",
					Password:   "secret",
				}
				c.Log = LogConfig{
					Level:     "debug",
					Format:    "json",
					AddSource: true,
				}
				c.RateLimit = RateLimitConfig{
					Rate:  50,
					Burst: 100,
				}
				c.HTTP = HTTPConfig{
					ReadTimeout:  5 * time.Second,
					WriteTimeout: 6 * time.Second,
					IdleTimeout:  90 * time.Second,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			setDefaults(v)
			v.SetConfigType(ConfigFileType)
			require.NoError(t, v.ReadConfig(bytes.NewBufferString(tt.yaml)))

			var got Config
			require.NoError(t, v.Unmarshal(&got))
			want := defaultConfig()
			tt.mutate(want)
			assert.Equal(t, *want, got)
		})
	}
}
