package stockbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// tokenKey is the Redis key under which the current access/refresh pair lives.
const tokenKey = "stockbit:tokens"

// TokenStore persists the access/refresh token pair so tokens survive restarts.
type TokenStore interface {
	Get(ctx context.Context) (*TokenData, error)
	Set(ctx context.Context, td *TokenData) error
}

// RedisTokenStore persists tokens in Redis.
type RedisTokenStore struct {
	client redis.Cmdable
}

func NewRedisTokenStore(client redis.Cmdable) *RedisTokenStore {
	return &RedisTokenStore{client: client}
}

func (s *RedisTokenStore) Get(ctx context.Context) (*TokenData, error) {
	raw, err := s.client.Get(ctx, tokenKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stockbit: load token: %w", err)
	}
	var td TokenData
	if err := json.Unmarshal(raw, &td); err != nil {
		return nil, fmt.Errorf("stockbit: decode token: %w", err)
	}
	return &td, nil
}

func (s *RedisTokenStore) Set(ctx context.Context, td *TokenData) error {
	raw, err := json.Marshal(td)
	if err != nil {
		return fmt.Errorf("stockbit: encode token: %w", err)
	}
	// Use the refresh token's expiry as the Redis TTL so stale tokens
	// are cleaned up automatically.
	ttl := time.Until(td.refreshExpiry())
	if ttl <= 0 {
		ttl = 24 * time.Hour // fallback when expiry is unknown
	}
	if err := s.client.Set(ctx, tokenKey, raw, ttl).Err(); err != nil {
		return fmt.Errorf("stockbit: save token: %w", err)
	}
	return nil
}

// parseExpiry parses an RFC3339 expiry string; empty or malformed yields the
// zero time, which callers treat as "unknown".
func parseExpiry(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (td *TokenData) accessExpiry() time.Time  { return parseExpiry(td.Access.ExpiredAt) }
func (td *TokenData) refreshExpiry() time.Time { return parseExpiry(td.Refresh.ExpiredAt) }
