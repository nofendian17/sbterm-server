// Package repository implements the repository contracts using PostgreSQL/Redis backends.

package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

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

func (s *RedisRefreshStore) ConsumeRefresh(ctx context.Context, jti string) (string, bool) {
	v, err := s.client.Get(ctx, refreshKeyPrefix+jti).Result()
	if errors.Is(err, redis.Nil) {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return v, true
}

func (s *RedisRefreshStore) DeleteRefresh(ctx context.Context, jti string) error {
	if err := s.client.Del(ctx, refreshKeyPrefix+jti).Err(); err != nil {
		return fmt.Errorf("refresh store: del %q: %w", jti, err)
	}
	return nil
}
