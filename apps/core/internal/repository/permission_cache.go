package repository

import (
	"context"
	"time"
)

//go:generate go run go.uber.org/mock/mockgen -source=permission_cache.go -destination=../mocks/mock_permission_cache.go -package=mocks -typed

// PermissionCache caches a user's resolved permission set in Redis to avoid
// repeated DB joins. The cache is invalidated whenever a role or permission
// assignment changes.
type PermissionCache interface {
	// Get returns the cached permission set for the given user, or nil if absent.
	Get(ctx context.Context, userID string) ([]string, bool)
	// Set stores the permission set for the given user with the specified TTL.
	Set(ctx context.Context, userID string, perms []string, ttl time.Duration) error
	// Invalidate removes the cached permission set for the given user.
	Invalidate(ctx context.Context, userID string) error
}
