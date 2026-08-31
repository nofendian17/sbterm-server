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

func (s *RedisRefreshStore) StoreRefresh(jti, userID string, ttl time.Duration) error {
	if err := s.client.Set(context.Background(), refreshKeyPrefix+jti, userID, ttl).Err(); err != nil {
		return fmt.Errorf("refresh store: set %q: %w", jti, err)
	}
	return nil
}

func (s *RedisRefreshStore) ConsumeRefresh(jti string) (string, bool) {
	v, err := s.client.Get(context.Background(), refreshKeyPrefix+jti).Result()
	if errors.Is(err, redis.Nil) {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return v, true
}

func (s *RedisRefreshStore) DeleteRefresh(jti string) error {
	if err := s.client.Del(context.Background(), refreshKeyPrefix+jti).Err(); err != nil {
		return fmt.Errorf("refresh store: del %q: %w", jti, err)
	}
	return nil
}
