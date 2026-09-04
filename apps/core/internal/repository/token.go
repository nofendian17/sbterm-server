package repository

import (
	"context"
	"time"
)

// RefreshStore persists refresh-token JTIs so they can be validated and
// consumed (single-use) by the auth usecase. The interface lives in the
// contract layer (internal/repository) so usecases depend on it without
// importing infrastructure.
type RefreshStore interface {
	// StoreRefresh persists the mapping refresh:<jti> -> userID with the
	// given TTL.
	StoreRefresh(ctx context.Context, jti, userID string, ttl time.Duration) error
	// ConsumeRefresh atomically reads the userID for a stored refresh JTI and
	// deletes the key. Returns the userID and true if the JTI existed, or
	// ("", false) if it was already consumed or expired. The atomic read+delete
	// prevents TOCTOU races on concurrent refresh attempts.
	ConsumeRefresh(ctx context.Context, jti string) (userID string, ok bool)
	// DeleteRefresh removes a stored jti (logout / rotation).
	DeleteRefresh(ctx context.Context, jti string) error
}
