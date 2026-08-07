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
	cfg := &redis.Options{}
	o := &options{}
	WithMaxRetries(5)(o)
	WithPoolSize(20)(o)
	WithMinIdleConns(2)(o)
	WithDialTimeout(2 * time.Second)(o)
	WithReadTimeout(time.Second)(o)
	WithWriteTimeout(1500 * time.Millisecond)(o)
	o.apply(cfg)

	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, 20, cfg.PoolSize)
	assert.Equal(t, 2, cfg.MinIdleConns)
	assert.Equal(t, 2*time.Second, cfg.DialTimeout)
	assert.Equal(t, time.Second, cfg.ReadTimeout)
	assert.Equal(t, 1500*time.Millisecond, cfg.WriteTimeout)
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
	server := miniredis.RunT(t)

	rdb, err := New(ctx, "redis://"+server.Addr()+"/0")
	require.NoError(t, err)
	defer rdb.Shutdown()

	assert.NoError(t, rdb.Ping(ctx))

	server.Close()
	assert.Error(t, rdb.Ping(ctx))
}

func TestHealthCheck(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)

	rdb, err := New(ctx, "redis://"+server.Addr()+"/0")
	require.NoError(t, err)
	defer rdb.Shutdown()

	assert.NoError(t, rdb.HealthCheck(ctx))

	server.Close()
	assert.Error(t, rdb.HealthCheck(ctx))
}

func TestShutdown(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)

	rdb, err := New(ctx, "redis://"+server.Addr()+"/0")
	require.NoError(t, err)

	assert.NoError(t, rdb.Shutdown())
	assert.Error(t, rdb.Ping(ctx))
}
