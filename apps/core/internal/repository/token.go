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
	// ConsumeRefresh returns the userID for a stored jti and whether it
	// existed. Later tasks may delete the jti on a successful consume.
	ConsumeRefresh(ctx context.Context, jti string) (userID string, ok bool)
	// DeleteRefresh removes a stored jti (logout / rotation).
	DeleteRefresh(ctx context.Context, jti string) error
}
