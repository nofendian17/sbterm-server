package hotstate

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T, ttl time.Duration) (*Store, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewStore(rdb, "ob", ttl), mr
}

func sampleUpdate() BookUpdate {
	return BookUpdate{
		Symbol:    "BBCA",
		Board:     "BOARD_TYPE_RG",
		Bid:       &Side{Seq: 41, Prices: []float64{7750, 7745}, Qtys: []int64{100, 50}},
		Ask:       &Side{Seq: 42, Prices: []float64{7760}, Qtys: []int64{200}},
		ReceiveTS: time.Date(2026, 8, 27, 10, 15, 3, 123456789, time.UTC),
	}
}

func TestStore_SetBook(t *testing.T) {
	ctx := context.Background()
	store, mr := newTestStore(t, 24*time.Hour)

	require.NoError(t, store.SetBook(ctx, sampleUpdate()))

	key := "ob:book:BBCA"
	assert.Equal(t, "7750", mr.HGet(key, "bid_px_0"))
	assert.Equal(t, "50", mr.HGet(key, "bid_qty_1"))
	assert.Equal(t, "7760", mr.HGet(key, "ask_px_0"))
	assert.Equal(t, "42", mr.HGet(key, "ask_seq"))
	assert.Equal(t, "BOARD_TYPE_RG", mr.HGet(key, "board"))

	gotTTL := mr.TTL(key)
	assert.GreaterOrEqual(t, gotTTL, 23*time.Hour, "the full write must apply the configured TTL")

	parsed, err := time.Parse(time.RFC3339Nano, mr.HGet(key, "ts_receive"))
	require.NoError(t, err)
	assert.Equal(t, sampleUpdate().ReceiveTS.UTC(), parsed.UTC())
}

func TestStore_TouchBook(t *testing.T) {
	ctx := context.Background()

	t.Run("creates a minimal liveness entry for unknown symbols", func(t *testing.T) {
		store, mr := newTestStore(t, 24*time.Hour)
		now := time.Date(2026, 8, 27, 16, 0, 0, 0, time.UTC)

		require.NoError(t, store.TouchBook(ctx, "GULA", now))

		assert.True(t, mr.Exists("ob:book:GULA"),
			"touch must keep the key alive even before any snapshot lands")
		assert.GreaterOrEqual(t, mr.TTL("ob:book:GULA"), 23*time.Hour)
	})

	t.Run("refreshes the ttl while keeping stored levels intact", func(t *testing.T) {
		store, mr := newTestStore(t, 24*time.Hour)
		require.NoError(t, store.SetBook(ctx, sampleUpdate()))

		// Simulate time pressure: shrink to 30 seconds, then touch.
		mr.SetTTL("ob:book:BBCA", 30*time.Second)
		later := time.Date(2026, 8, 27, 11, 0, 0, 0, time.UTC)
		require.NoError(t, store.TouchBook(ctx, "BBCA", later))

		assert.GreaterOrEqual(t, mr.TTL("ob:book:BBCA"), 23*time.Hour,
			"touch must restore the full ttl")
		assert.Equal(t, strconv.FormatFloat(7750, 'f', -1, 64), mr.HGet("ob:book:BBCA", "bid_px_0"),
			"touch must not clobber stored levels")
	})
}

func TestStore_HashDedup(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t, 24*time.Hour)

	seen, err := store.SeenBefore(ctx, "BBCA", "abc123")
	require.NoError(t, err)
	assert.False(t, seen)

	require.NoError(t, store.MarkSeen(ctx, "BBCA", "abc123"))

	seen, err = store.SeenBefore(ctx, "BBCA", "abc123")
	require.NoError(t, err)
	assert.True(t, seen, "a marked body hash must be recognized across restarts")

	unseen, err := store.SeenBefore(ctx, "BBCA", "different")
	require.NoError(t, err)
	assert.False(t, unseen)
}
