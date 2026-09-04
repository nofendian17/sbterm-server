// Package repository implements the repository contracts using PostgreSQL/Redis backends.

package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// consumeScript atomically GETs the value and DELetes the key in a single
// round-trip, preventing a TOCTOU race between separate GET and DEL calls.
var consumeScript = redis.NewScript(`
	local v = redis.call('GET', KEYS[1])
	if v then
		redis.call('DEL', KEYS[1])
		return v
	end
	return nil
`)

// refreshKeyPrefix is the Redis key prefix for stored refresh JTIs.
const refreshKeyPrefix = "refresh:"

// NewRedisRefreshStore builds a Redis-backed RefreshStore. The RefreshStore
// interface itself is declared in the contract package internal/repository.

type RedisRefreshStore struct {
	client *redis.Client
}

// NewRedisRefreshStore builds a Redis-backed RefreshStore.
func NewRedisRefreshStore(client *redis.Client) *RedisRefreshStore {
	return &RedisRefreshStore{client: client}
}

func (s *RedisRefreshStore) StoreRefresh(ctx context.Context, jti, userID string, ttl time.Duration) error {
	if err := s.client.Set(
		ctx,
		refreshKeyPrefix+jti,
		userID,
		ttl,
	).Err(); err != nil {
		return fmt.Errorf("refresh store: set %q: %w", jti, err)
	}
	return nil
}

// ConsumeRefresh atomically reads the userID for a stored refresh JTI and
// deletes the key in a single Redis round-trip. This prevents a TOCTOU race
// where concurrent requests could both pass a GET check before either's DEL
// completes, defeating single-use rotation.
func (s *RedisRefreshStore) ConsumeRefresh(ctx context.Context, jti string) (string, bool) {
	result, err := consumeScript.Run(ctx, s.client, []string{refreshKeyPrefix + jti}).Result()
	if errors.Is(err, redis.Nil) {
		return "", false
	}
	if err != nil {
		return "", false
	}
	userID, ok := result.(string)
	return userID, ok
}

func (s *RedisRefreshStore) DeleteRefresh(ctx context.Context, jti string) error {
	if err := s.client.Del(ctx, refreshKeyPrefix+jti).Err(); err != nil {
		return fmt.Errorf("refresh store: del %q: %w", jti, err)
	}
	return nil
}
