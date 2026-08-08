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

func TestRedisTokenStore(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		setup func(t *testing.T, store *RedisTokenStore)
		want  *TokenData
	}{
		{
			name: "round trips a token",
			setup: func(t *testing.T, store *RedisTokenStore) {
				require.NoError(t, store.Set(ctx, &TokenData{
					Access:  TokenPair{Token: "at1", ExpiredAt: "2026-01-01T00:00:00Z"},
					Refresh: TokenPair{Token: "rt1", ExpiredAt: "2026-02-01T00:00:00Z"},
				}))
			},
			want: &TokenData{
				Access:  TokenPair{Token: "at1", ExpiredAt: "2026-01-01T00:00:00Z"},
				Refresh: TokenPair{Token: "rt1", ExpiredAt: "2026-02-01T00:00:00Z"},
			},
		},
		{
			name: "returns nil when empty",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, _ := newTestStore(t)
			if tt.setup != nil {
				tt.setup(t, store)
			}

			got, err := store.Get(ctx)
			require.NoError(t, err)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseExpiry(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{name: "parses a valid timestamp", input: "2026-01-01T00:00:00Z", want: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{name: "empty string returns zero time", input: "", want: time.Time{}},
		{name: "invalid string returns zero time", input: "garbage", want: time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseExpiry(tt.input))
		})
	}
}
