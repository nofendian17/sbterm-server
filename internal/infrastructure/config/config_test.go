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
			BaseURL:                   "https://exodus.stockbit.com",
			Timeout:                   30 * time.Second,
			RetryCount:                3,
			WSURL:                     "wss://wssfeed.stockbit.com/",
			WSEnabled:                 false,
			WSPingInterval:            30 * time.Second,
			WSReconnectBackoffInitial: time.Second,
			WSReconnectBackoffMax:     30 * time.Second,
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
				"APP_NAME":                                  "sbterm",
				"APP_VERSION":                               "1.2.3",
				"APP_PORT":                                  ":9090",
				"APP_DATABASE_URL":                          "postgres://u:p@host:5432/db",
				"APP_DATABASE_MAX_CONNS":                    "25",
				"APP_DATABASE_MIN_CONNS":                    "2",
				"APP_DATABASE_MAX_CONN_IDLE_TIME":           "1m",
				"APP_DATABASE_MAX_CONN_LIFETIME":            "10m",
				"APP_REDIS_URL":                             "redis://cache:6379/1",
				"APP_REDIS_MAX_RETRIES":                     "5",
				"APP_REDIS_POOL_SIZE":                       "20",
				"APP_REDIS_MIN_IDLE_CONNS":                  "2",
				"APP_REDIS_DIAL_TIMEOUT":                    "2s",
				"APP_REDIS_READ_TIMEOUT":                    "1s",
				"APP_REDIS_WRITE_TIMEOUT":                   "1500ms",
				"APP_STOCKBIT_BASE_URL":                     "https://exodus.example.com",
				"APP_STOCKBIT_TIMEOUT":                      "15s",
				"APP_STOCKBIT_RETRY_COUNT":                  "1",
				"APP_STOCKBIT_PLAYER_ID":                    "p123",
				"APP_STOCKBIT_USERNAME":                     "budi",
				"APP_STOCKBIT_PASSWORD":                     "secret",
				"APP_STOCKBIT_WS_URL":                       "wss://wssfeed.example.com/",
				"APP_STOCKBIT_WS_ENABLED":                   "true",
				"APP_STOCKBIT_WS_PING_INTERVAL":             "60s",
				"APP_STOCKBIT_WS_RECONNECT_BACKOFF_INITIAL": "2s",
				"APP_STOCKBIT_WS_RECONNECT_BACKOFF_MAX":     "60s",
				"APP_LOG_LEVEL":                             "debug",
				"APP_LOG_FORMAT":                            "json",
				"APP_LOG_ADD_SOURCE":                        "true",
				"APP_RATE_LIMIT_RATE":                       "50",
				"APP_RATE_LIMIT_BURST":                      "100",
				"APP_HTTP_READ_TIMEOUT":                     "5s",
				"APP_HTTP_WRITE_TIMEOUT":                    "6s",
				"APP_HTTP_IDLE_TIMEOUT":                     "90s",
			},
			want: &Config{
				App: AppConfig{
					Name:    "sbterm",
					Version: "1.2.3",
				},
				Port: ":9090",
				Database: DatabaseConfig{
					URL:             "postgres://u:p@host:5432/db",
					MaxConns:        25,
					MinConns:        2,
					MaxConnIdleTime: 1 * time.Minute,
					MaxConnLifetime: 10 * time.Minute,
				},
				Redis: RedisConfig{
					URL:          "redis://cache:6379/1",
					MaxRetries:   5,
					PoolSize:     20,
					MinIdleConns: 2,
					DialTimeout:  2 * time.Second,
					ReadTimeout:  1 * time.Second,
					WriteTimeout: 1500 * time.Millisecond,
				},
				Stockbit: StockbitConfig{
					BaseURL:                   "https://exodus.example.com",
					Timeout:                   15 * time.Second,
					RetryCount:                1,
					PlayerID:                  "p123",
					Username:                  "budi",
					Password:                  "secret",
					WSURL:                     "wss://wssfeed.example.com/",
					WSEnabled:                 true,
					WSPingInterval:            60 * time.Second,
					WSReconnectBackoffInitial: 2 * time.Second,
					WSReconnectBackoffMax:     60 * time.Second,
				},
				Log: LogConfig{
					Level:     "debug",
					Format:    "json",
					AddSource: true,
				},
				RateLimit: RateLimitConfig{
					Rate:  50,
					Burst: 100,
				},
				HTTP: HTTPConfig{
					ReadTimeout:  5 * time.Second,
					WriteTimeout: 6 * time.Second,
					IdleTimeout:  90 * time.Second,
				},
			},
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
			yaml: "app:\n  name: custom-app\n  version: 2.0.0\nport: \":9090\"\ndatabase:\n  max_conns: 42\nredis:\n  pool_size: 25\nlog:\n  format: json\nstockbit:\n  ws_enabled: true\n  ws_symbols:\n    - BBCA\n    - BBRI\n",
			mutate: func(c *Config) {
				c.App.Name = "custom-app"
				c.App.Version = "2.0.0"
				c.Port = ":9090"
				c.Database.MaxConns = 42
				c.Redis.PoolSize = 25
				c.Log.Format = "json"
				c.Stockbit.WSEnabled = true
				c.Stockbit.WSSymbols = []string{"BBCA", "BBRI"}
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

			var got Config
			require.NoError(t, v.Unmarshal(&got))
			want := defaultConfig()
			tt.mutate(want)
			assert.Equal(t, *want, got)
		})
	}
}
