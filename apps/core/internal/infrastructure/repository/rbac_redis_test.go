package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	srv, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(srv.Close)
	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { client.Close() })
	return client
}

func TestPermissionCache(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		perms     []string
		ttl       time.Duration
		setup     func(t *testing.T, cache *RedisPermissionCache)
		wantOK    bool
		wantPerms []string
	}{
		{
			name:   "get miss before set",
			userID: "u1",
			setup:  func(t *testing.T, cache *RedisPermissionCache) {},
			wantOK: false,
		},
		{
			name:   "set and get hit",
			userID: "u1",
			perms:  []string{"profile:read", "watchlist:write"},
			ttl:    5 * time.Minute,
			setup: func(t *testing.T, cache *RedisPermissionCache) {
				require.NoError(t, cache.Set(context.Background(), "u1", []string{"profile:read", "watchlist:write"}, 5*time.Minute))
			},
			wantOK:    true,
			wantPerms: []string{"profile:read", "watchlist:write"},
		},
		{
			name:   "invalidate removes cached entry",
			userID: "u1",
			perms:  []string{"a"},
			ttl:    5 * time.Minute,
			setup: func(t *testing.T, cache *RedisPermissionCache) {
				require.NoError(t, cache.Set(context.Background(), "u1", []string{"a"}, 5*time.Minute))
				require.NoError(t, cache.Invalidate(context.Background(), "u1"))
			},
			wantOK: false,
		},
		{
			name:   "empty permissions set",
			userID: "u1",
			perms:  []string{},
			ttl:    5 * time.Minute,
			setup: func(t *testing.T, cache *RedisPermissionCache) {
				require.NoError(t, cache.Set(context.Background(), "u1", []string{}, 5*time.Minute))
			},
			wantOK:    true,
			wantPerms: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestRedisClient(t)
			cache := NewRedisPermissionCache(client)
			tt.setup(t, cache)

			got, ok := cache.Get(context.Background(), tt.userID)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantPerms, got)
			}
		})
	}
}
