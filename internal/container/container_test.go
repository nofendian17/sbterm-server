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
		Port:              ":0",
		DatabaseURL:       databaseURL,
		DBMaxConns:        10,
		DBMinConns:        0,
		DBMaxConnLifetime: 30 * time.Minute,
		DBMaxConnIdleTime: 5 * time.Minute,
		RedisURL:          redisURL,
		RedisMaxRetries:   1,
		RedisPoolSize:     1,
		RedisMinIdleConns: 0,
		RedisDialTimeout:  time.Second,
		RedisReadTimeout:  time.Second,
		RedisWriteTimeout: time.Second,
		LogLevel:          "info",
		LogFormat:         "text",
		RateLimitRate:     100,
		RateLimitBurst:    200,
		HTTPReadTimeout:   10 * time.Second,
		HTTPWriteTimeout:  10 * time.Second,
		HTTPIdleTimeout:   60 * time.Second,
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
			name: "dead database still serves health 200 with database down",
			cfg:  testConfig("postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1", testRedisURL(t)),
			check: func(t *testing.T, srv *deliveryhttp.Server) {
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/health", nil)
				srv.Handler().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusOK, rec.Code)

				var env struct {
					Data struct {
						Status   string `json:"status"`
						Database string `json:"database"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
				assert.Equal(t, "ok", env.Data.Status)
				assert.Equal(t, "down", env.Data.Database)
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
			injector := New(tt.cfg, logger)

			srv, err := do.Invoke[*deliveryhttp.Server](injector)
			if tt.wantBuildErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.check(t, srv)
		})
	}
}

func TestShutdownReportIncludesServices(t *testing.T) {
	logger := log.New(log.WithWriter(io.Discard))
	injector := New(testConfig("postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1", testRedisURL(t)), logger)

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
