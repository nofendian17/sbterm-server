package repository

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestRedisRefreshStore_StoreThenConsume(t *testing.T) {
	client := newTestRedis(t)
	store := NewRedisRefreshStore(client)

	const (
		jti    = "abc123"
		userID = "u1"
		ttl    = 10 * time.Minute
	)

	require.NoError(t, store.StoreRefresh(jti, userID, ttl))

	got, ok := store.ConsumeRefresh(jti)
	require.True(t, ok)
	require.Equal(t, userID, got)
}

func TestRedisRefreshStore_DeleteThenConsume(t *testing.T) {
	client := newTestRedis(t)
	store := NewRedisRefreshStore(client)

	const (
		jti    = "def456"
		userID = "u2"
		ttl    = 10 * time.Minute
	)

	require.NoError(t, store.StoreRefresh(jti, userID, ttl))
	require.NoError(t, store.DeleteRefresh(jti))

	_, ok := store.ConsumeRefresh(jti)
	require.False(t, ok)
}

func TestRedisRefreshStore_ConsumeMissing(t *testing.T) {
	client := newTestRedis(t)
	store := NewRedisRefreshStore(client)

	_, ok := store.ConsumeRefresh("nope")
	require.False(t, ok)
}
