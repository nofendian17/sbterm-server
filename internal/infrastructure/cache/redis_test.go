package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsApply(t *testing.T) {
	tests := []struct {
		name string
		opt  Option
		want func(t *testing.T, cfg *redis.Options)
	}{
		{name: "max retries", opt: WithMaxRetries(5), want: func(t *testing.T, cfg *redis.Options) { assert.Equal(t, 5, cfg.MaxRetries) }},
		{name: "pool size", opt: WithPoolSize(20), want: func(t *testing.T, cfg *redis.Options) { assert.Equal(t, 20, cfg.PoolSize) }},
		{name: "min idle conns", opt: WithMinIdleConns(2), want: func(t *testing.T, cfg *redis.Options) { assert.Equal(t, 2, cfg.MinIdleConns) }},
		{name: "dial timeout", opt: WithDialTimeout(2 * time.Second), want: func(t *testing.T, cfg *redis.Options) { assert.Equal(t, 2*time.Second, cfg.DialTimeout) }},
		{name: "read timeout", opt: WithReadTimeout(time.Second), want: func(t *testing.T, cfg *redis.Options) { assert.Equal(t, time.Second, cfg.ReadTimeout) }},
		{name: "write timeout", opt: WithWriteTimeout(1500 * time.Millisecond), want: func(t *testing.T, cfg *redis.Options) { assert.Equal(t, 1500*time.Millisecond, cfg.WriteTimeout) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &redis.Options{}
			o := &options{}
			tt.opt(o)
			o.apply(cfg)
			tt.want(t, cfg)
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "valid url", url: "redis://localhost:6379/0"},
		{name: "malformed url", url: "://broken", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rdb, err := New(context.Background(), tt.url)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, rdb)
			assert.NoError(t, rdb.Shutdown())
		})
	}
}

func TestPing(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		serverOpen bool
	}{
		{name: "ping succeeds", serverOpen: true},
		{name: "ping fails after server closes", serverOpen: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := miniredis.RunT(t)
			defer server.Close()

			rdb, err := New(ctx, "redis://"+server.Addr()+"/0")
			require.NoError(t, err)
			defer rdb.Shutdown()

			if !tt.serverOpen {
				server.Close()
				assert.Error(t, rdb.Ping(ctx))
				return
			}
			assert.NoError(t, rdb.Ping(ctx))
		})
	}
}

func TestHealthCheck(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		serverOpen bool
	}{
		{name: "health check succeeds", serverOpen: true},
		{name: "health check fails after server closes", serverOpen: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := miniredis.RunT(t)
			defer server.Close()

			rdb, err := New(ctx, "redis://"+server.Addr()+"/0")
			require.NoError(t, err)
			defer rdb.Shutdown()

			if !tt.serverOpen {
				server.Close()
				assert.Error(t, rdb.HealthCheck(ctx))
				return
			}
			assert.NoError(t, rdb.HealthCheck(ctx))
		})
	}
}

func TestShutdown(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
	}{
		{name: "shutdown stops further pings"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := miniredis.RunT(t)

			rdb, err := New(ctx, "redis://"+server.Addr()+"/0")
			require.NoError(t, err)

			assert.NoError(t, rdb.Shutdown())
			assert.Error(t, rdb.Ping(ctx))
		})
	}
}
