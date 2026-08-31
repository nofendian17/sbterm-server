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

func TestPermissionCache_SetAndGet(t *testing.T) {
	srv, err := miniredis.Run()
	require.NoError(t, err)
	defer srv.Close()

	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	defer client.Close()

	cache := NewRedisPermissionCache(client)
	ctx := context.Background()
	perms := []string{"profile:read", "watchlist:write"}

	// Get misses before set
	got, ok := cache.Get(ctx, "u1")
	assert.False(t, ok)
	assert.Nil(t, got)

	// Set
	require.NoError(t, cache.Set(ctx, "u1", perms, 5*time.Minute))

	// Get hits
	got, ok = cache.Get(ctx, "u1")
	assert.True(t, ok)
	assert.Equal(t, perms, got)
}

func TestPermissionCache_Invalidate(t *testing.T) {
	srv, err := miniredis.Run()
	require.NoError(t, err)
	defer srv.Close()

	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	defer client.Close()

	cache := NewRedisPermissionCache(client)
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "u1", []string{"a"}, 5*time.Minute))
	got, ok := cache.Get(ctx, "u1")
	assert.True(t, ok)
	assert.Equal(t, []string{"a"}, got)

	require.NoError(t, cache.Invalidate(ctx, "u1"))
	got, ok = cache.Get(ctx, "u1")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestPermissionCache_EmptyPerms(t *testing.T) {
	srv, err := miniredis.Run()
	require.NoError(t, err)
	defer srv.Close()

	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	defer client.Close()

	cache := NewRedisPermissionCache(client)
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "u1", []string{}, 5*time.Minute))
	got, ok := cache.Get(ctx, "u1")
	assert.True(t, ok)
	assert.Equal(t, []string{}, got)
}
