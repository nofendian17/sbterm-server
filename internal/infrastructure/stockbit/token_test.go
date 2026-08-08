package stockbit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*RedisTokenStore, *miniredis.Miniredis) {
	t.Helper()
	srv := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { client.Close() })
	return NewRedisTokenStore(client), srv
}

func TestRedisTokenStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)

	td := &TokenData{
		Access:  TokenPair{Token: "at1", ExpiredAt: "2026-01-01T00:00:00Z"},
		Refresh: TokenPair{Token: "rt1", ExpiredAt: "2026-02-01T00:00:00Z"},
	}
	require.NoError(t, store.Set(ctx, td))

	got, err := store.Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, td, got)
}

func TestRedisTokenStoreGetEmpty(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)

	got, err := store.Get(ctx)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestParseExpiry(t *testing.T) {
	got := parseExpiry("2026-01-01T00:00:00Z")
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), got)
	assert.True(t, parseExpiry("").IsZero())
	assert.True(t, parseExpiry("garbage").IsZero())
}
