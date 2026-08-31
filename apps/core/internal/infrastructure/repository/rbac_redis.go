package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const permCacheKeyPrefix = "perms:"

// redisPermissionCache is the Redis-backed implementation of repository.PermissionCache.
type RedisPermissionCache struct {
	client *redis.Client
}

// NewRedisPermissionCache builds a PermissionCache backed by the given Redis client.
func NewRedisPermissionCache(client *redis.Client) *RedisPermissionCache {
	return &RedisPermissionCache{client: client}
}

// Get returns the cached permission set for the given user.
func (c *RedisPermissionCache) Get(ctx context.Context, userID string) ([]string, bool) {
	data, err := c.client.Get(ctx, permCacheKeyPrefix+userID).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	var perms []string
	if err := json.Unmarshal(data, &perms); err != nil {
		return nil, false
	}
	return perms, true
}

// Set stores the permission set for the given user with the specified TTL.
func (c *RedisPermissionCache) Set(ctx context.Context, userID string, perms []string, ttl time.Duration) error {
	data, err := json.Marshal(perms)
	if err != nil {
		return fmt.Errorf("perm cache set: marshal: %w", err)
	}
	if err := c.client.Set(ctx, permCacheKeyPrefix+userID, data, ttl).Err(); err != nil {
		return fmt.Errorf("perm cache set: %w", err)
	}
	return nil
}

// Invalidate removes the cached permission set for the given user.
func (c *RedisPermissionCache) Invalidate(ctx context.Context, userID string) error {
	if err := c.client.Del(ctx, permCacheKeyPrefix+userID).Err(); err != nil {
		return fmt.Errorf("perm cache invalidate: %w", err)
	}
	return nil
}
