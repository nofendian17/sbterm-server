package container

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	deliveryhttp "github.com/nofendian17/sbterm/apps/api/internal/delivery/http"
	"github.com/nofendian17/sbterm/apps/api/internal/infrastructure/config"
	"github.com/nofendian17/sbterm/libs/pkg/log"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

func testConfig(redisURL string) *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name:    "test-app",
			Version: "1.0.0",
		},
		Port: ":9999",
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
			name:         "malformed redis url fails to build server",
			cfg:          testConfig("://broken"),
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
			if tt.check != nil {
				tt.check(t, srv)
			}
		})
	}
}

func TestPingInfraFailsWhenRedisUnreachable(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr string
	}{
		{
			name:    "dead redis makes ping infra fail",
			cfg:     testConfig("redis://127.0.0.1:1/0?dial_timeout=1ms"),
			wantErr: "redis unreachable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.New(log.WithWriter(io.Discard))
			injector := New(tt.cfg, logger)

			err := pingInfra(injector)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestShutdownReportIncludesServices(t *testing.T) {
	tests := []struct {
		name  string
		cfg   *config.Config
		wants []struct {
			service string
			msg     string
		}
	}{
		{
			name: "shutdown report includes http server and redis",
			cfg:  testConfig(testRedisURL(t)),
			wants: []struct {
				service string
				msg     string
			}{
				{service: "delivery/http.Server", msg: "shutdown report should include the http server, got: %v"},
				{service: "cache.Redis", msg: "shutdown report should include redis, got: %v"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.New(log.WithWriter(io.Discard))
			injector := New(tt.cfg, logger)

			srv, err := do.Invoke[*deliveryhttp.Server](injector)
			require.NoError(t, err)
			assert.NotNil(t, srv)

			report := injector.ShutdownWithContext(context.Background())
			assert.True(t, report.Succeed, "shutdown errors: %v", report.Errors)

			names := make([]string, 0, len(report.Services))
			for _, svc := range report.Services {
				names = append(names, svc.Service)
			}
			for _, want := range tt.wants {
				assert.True(t, containsService(names, want.service), want.msg, names)
			}
		})
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}

func waitHTTPReady(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready", addr)
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr string
	}{
		{
			name: "valid level and format",
			cfg:  &config.Config{Log: config.LogConfig{Level: "debug", Format: "json", AddSource: true}},
		},
		{
			name:    "invalid log level",
			cfg:     &config.Config{Log: config.LogConfig{Level: "verbose", Format: "text"}},
			wantErr: "invalid level",
		},
		{
			name:    "invalid log format",
			cfg:     &config.Config{Log: config.LogConfig{Level: "info", Format: "yaml"}},
			wantErr: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := newLogger(tt.cfg)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, logger)
		})
	}
}

func TestAwaitShutdown(t *testing.T) {
	logger := log.New(log.WithWriter(io.Discard))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	tests := []struct {
		name  string
		setup func(t *testing.T) (*deliveryhttp.Server, *do.RootScope, string)
		stop  func(t *testing.T, addr string)
	}{
		{
			name: "returns error when server fails to start",
			setup: func(t *testing.T) (*deliveryhttp.Server, *do.RootScope, string) {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				t.Cleanup(func() { ln.Close() })
				addr := ln.Addr().String()
				srv := deliveryhttp.NewServer(handler, deliveryhttp.WithAddr(addr))
				return srv, do.New(), addr
			},
		},
		{
			name: "stops and returns after SIGTERM",
			setup: func(t *testing.T) (*deliveryhttp.Server, *do.RootScope, string) {
				addr := freeAddr(t)
				srv := deliveryhttp.NewServer(handler, deliveryhttp.WithAddr(addr))
				return srv, do.New(), addr
			},
			stop: func(t *testing.T, addr string) {
				waitHTTPReady(t, addr)
				require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGTERM))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, injector, addr := tt.setup(t)
			if tt.stop == nil {
				err := awaitShutdown(srv, injector, logger)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "http server failed")
				return
			}

			done := make(chan error, 1)
			go func() { done <- awaitShutdown(srv, injector, logger) }()

			tt.stop(t, addr)
			err := <-done
			require.NoError(t, err)
		})
	}
}

func containsService(names []string, suffix string) bool {
	for _, n := range names {
		if strings.HasSuffix(n, suffix) {
			return true
		}
	}
	return false
}

func TestStockbitClientIsAuthenticatedWhenResolvedFirst(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "resolving client directly matches refresher client",
			cfg:  testConfig(testRedisURL(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.New(log.WithWriter(io.Discard))
			injector := New(tt.cfg, logger)

			client, err := do.Invoke[*stockbit.Client](injector)
			require.NoError(t, err)
			require.NotNil(t, client)

			refresher, err := do.Invoke[*stockbit.Refresher](injector)
			require.NoError(t, err)
			require.NotNil(t, refresher)

			assert.Same(t, client, refresher.Client(), "resolving client directly must return the same authenticated instance as refresher.Client()")
		})
	}
}
