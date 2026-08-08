package container

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	deliveryhttp "github.com/nofendian17/sbterm-server/internal/delivery/http"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/config"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

func testConfig(databaseURL, redisURL string) *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name:    "test-app",
			Version: "1.0.0",
		},
		Port: ":9999",
		Database: config.DatabaseConfig{
			URL:             databaseURL,
			MaxConns:        10,
			MinConns:        0,
			MaxConnLifetime: 30 * time.Minute,
			MaxConnIdleTime: 5 * time.Minute,
		},
		Redis: config.RedisConfig{
			URL:          redisURL,
			MaxRetries:   1,
			PoolSize:     1,
			MinIdleConns: 0,
			DialTimeout:  time.Second,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
		},
		Log: config.LogConfig{
			Level:     "info",
			Format:    "text",
			AddSource: false,
		},
		RateLimit: config.RateLimitConfig{
			Rate:  100,
			Burst: 200,
		},
		HTTP: config.HTTPConfig{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func testRedisURL(t *testing.T) string {
	server := miniredis.RunT(t)
	return fmt.Sprintf("redis://%s/0", server.Addr())
}

func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.Config
		wantBuildErr bool
		check        func(t *testing.T, srv *deliveryhttp.Server)
	}{
		{
			name: "dead database returns 503 degraded with database down",
			cfg:  testConfig("postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1", testRedisURL(t)),
			check: func(t *testing.T, srv *deliveryhttp.Server) {
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/health", nil)
				srv.Handler().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

				var env struct {
					Data struct {
						Status   string `json:"status"`
						Database string `json:"database"`
						Redis    string `json:"redis"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
				assert.Equal(t, "degraded", env.Data.Status)
				assert.Equal(t, "down", env.Data.Database)
				assert.Equal(t, "up", env.Data.Redis)
			},
		},
		{
			name:         "malformed database url fails to build server",
			cfg:          testConfig("://broken", testRedisURL(t)),
			wantBuildErr: true,
		},
		{
			name:         "malformed redis url fails to build server",
			cfg:          testConfig("postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1", "://broken"),
			wantBuildErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.New(log.WithWriter(io.Discard))
			injector, err := New(tt.cfg, logger)
			if tt.wantBuildErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			srv, err := do.Invoke[*deliveryhttp.Server](injector)
			require.NoError(t, err)
			tt.check(t, srv)
		})
	}
}

func TestShutdownReportIncludesServices(t *testing.T) {
	logger := log.New(log.WithWriter(io.Discard))
	injector, err := New(testConfig("postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1", testRedisURL(t)), logger)
	require.NoError(t, err)

	srv, err := do.Invoke[*deliveryhttp.Server](injector)
	require.NoError(t, err)
	assert.NotNil(t, srv)

	report := injector.ShutdownWithContext(context.Background())
	assert.True(t, report.Succeed, "shutdown errors: %v", report.Errors)

	names := make([]string, 0, len(report.Services))
	for _, svc := range report.Services {
		names = append(names, svc.Service)
	}
	assert.True(t, containsService(names, "delivery/http.Server"),
		"shutdown report should include the http server, got: %v", names)
	assert.True(t, containsService(names, "database.Postgres"),
		"shutdown report should include the database, got: %v", names)
	assert.True(t, containsService(names, "cache.Redis"),
		"shutdown report should include redis, got: %v", names)
}

func containsService(names []string, suffix string) bool {
	for _, n := range names {
		if strings.HasSuffix(n, suffix) {
			return true
		}
	}
	return false
}
