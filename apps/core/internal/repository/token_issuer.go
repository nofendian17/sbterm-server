package repository

import (
	"context"
	"time"
)

// TokenIssuer issues and verifies signed access/refresh token pairs.
// The infrastructure implementation (JWT) lives behind this port so usecases
// never import crypto/signing libraries.
type TokenIssuer interface {
	// GenerateTokenPair issues a signed access token and a signed refresh
	// token for userID.
	GenerateTokenPair(ctx context.Context, userID string, expiresAt *time.Time) (access, refresh string, err error)
	// VerifyRefresh verifies a refresh token and returns the subject userID
	// and the token jti.
	VerifyRefresh(token string) (userID, jti string, err error)
	// ConsumeRefresh atomically reads the userID for a stored refresh JTI and
	// deletes the key. See repository.RefreshStore for semantics.
	ConsumeRefresh(ctx context.Context, jti string) (userID string, ok bool)
	// DeleteRefresh removes a stored refresh jti (logout / rotation).
	DeleteRefresh(ctx context.Context, jti string) error
}
