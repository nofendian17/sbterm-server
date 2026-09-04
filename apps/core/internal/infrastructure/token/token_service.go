// Package token implements JWT access/refresh token issuance backed by a
// RefreshStore.

package token

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// Claim type discriminators for access vs refresh tokens.
const (
	claimTypeAccess  = "access"
	claimTypeRefresh = "refresh"
)

// tokenClaims embeds RegisteredClaims and adds the custom `typ` claim that
// distinguishes an access token from a refresh token.
type tokenClaims struct {
	jwt.RegisteredClaims
	Typ string `json:"typ"`
}

// TokenService issues and verifies signed access/refresh token pairs. The
// interface keeps callers (and tests) decoupled from the JWT implementation.
type TokenService interface {
	// Sign issues a signed access JWT and a signed refresh JWT for userID.
	Sign(ctx context.Context, userID string, expiresAt *time.Time) (access, refresh string, err error)
	// VerifyAccess verifies an access JWT and returns the userID.
	VerifyAccess(token string) (userID string, err error)
	// VerifyRefresh verifies a refresh JWT and returns the userID and the token jti.
	VerifyRefresh(token string) (userID, jti string, err error)
}

// JWTTokenService is the JWT-based implementation of TokenService. It issues
// and verifies signed access/refresh pairs and persists refresh JTIs through
// a RefreshStore.
type JWTTokenService struct {
	secret       string
	accessTTL    time.Duration
	refreshTTL   time.Duration
	refreshStore repository.RefreshStore
}

var _ TokenService = (*JWTTokenService)(nil)
var _ repository.TokenIssuer = (*JWTTokenService)(nil)

// NewJWTTokenService builds a JWTTokenService.
//
// The expiresAt *time.Time parameter of Sign is reserved (per-user expiry is
// enforced server-side in a later task) and currently ignored; the signature
// is kept for forward compatibility.
func NewJWTTokenService(secret string, accessTTL, refreshTTL time.Duration, store repository.RefreshStore) *JWTTokenService {
	return &JWTTokenService{
		secret:       secret,
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		refreshStore: store,
	}
}

// Sign implements TokenService.
func (s *JWTTokenService) Sign(ctx context.Context, userID string, _ *time.Time) (access, refresh string, err error) {
	now := time.Now()

	access, err = s.signToken(
		userID,
		claimTypeAccess,
		s.accessTTL,
		now,
	)
	if err != nil {
		return "", "", err
	}

	refresh, err = s.signToken(
		userID,
		claimTypeRefresh,
		s.refreshTTL,
		now,
	)
	if err != nil {
		return "", "", err
	}

	// Refresh jti is persisted so the auth usecase can verify presence.
	if err := s.refreshStore.StoreRefresh(
		ctx,
		refreshJTIFrom(refresh),
		userID,
		s.refreshTTL,
	); err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// GenerateTokenPair is an alias for Sign. It exists so the JWTTokenService
// continues to satisfy repository.TokenIssuer (which is the wider port the
// auth usecase depends on). New code should call Sign.
func (s *JWTTokenService) GenerateTokenPair(ctx context.Context, userID string, expiresAt *time.Time) (access, refresh string, err error) {
	return s.Sign(ctx, userID, expiresAt)
}

// VerifyAccess verifies a signed access JWT (correct secret, HS256, valid
// typ, not expired) and returns the subject userID.
func (s *JWTTokenService) VerifyAccess(token string) (string, error) {
	claims, err := s.parse(token, claimTypeAccess)
	if err != nil {
		return "", err
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return "", errors.New("token expired")
	}
	return claims.Subject, nil
}

// VerifyRefresh verifies a signed refresh JWT (correct secret, HS256, valid
// typ="refresh", not expired) and returns the subject userID and the refresh
// jti. The auth usecase uses the jti to consume/rotate/delete the refresh
// token in the RefreshStore.
func (s *JWTTokenService) VerifyRefresh(token string) (userID, jti string, err error) {
	claims, err := s.parse(token, claimTypeRefresh)
	if err != nil {
		return "", "", err
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return "", "", errors.New("token expired")
	}
	return claims.Subject, claims.ID, nil
}

// StoreRefresh persists a refresh jti. See repository.RefreshStore.
func (s *JWTTokenService) StoreRefresh(ctx context.Context, jti, userID string, ttl time.Duration) error {
	return s.refreshStore.StoreRefresh(ctx, jti, userID, ttl)
}

// ConsumeRefresh atomically reads the userID for a stored refresh jti and
// deletes the key. See repository.RefreshStore for semantics.
func (s *JWTTokenService) ConsumeRefresh(ctx context.Context, jti string) (string, bool) {
	return s.refreshStore.ConsumeRefresh(ctx, jti)
}

// DeleteRefresh removes a stored refresh jti. See repository.RefreshStore.
func (s *JWTTokenService) DeleteRefresh(ctx context.Context, jti string) error {
	return s.refreshStore.DeleteRefresh(ctx, jti)
}

func (s *JWTTokenService) signToken(userID, typ string, ttl time.Duration, now time.Time) (string, error) {
	claims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        newJTI(),
		},
		Typ: typ,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString([]byte(s.secret))
	if err != nil {
		return "", fmt.Errorf("token: sign: %w", err)
	}
	return signed, nil
}

func (s *JWTTokenService) parse(token, wantTyp string) (*tokenClaims, error) {
	claims := &tokenClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims.Typ != wantTyp {
		return nil, fmt.Errorf("unexpected token type %q", claims.Typ)
	}
	return claims, nil
}

// refreshJTIFrom extracts the jti (claim "jti") from a signed refresh token.
func refreshJTIFrom(refresh string) string {
	claims := &tokenClaims{}
	// Parse without verification is fine here: the value is only used as a
	// store key and will be verified later by the auth usecase.
	_, _, err := jwt.NewParser().ParseUnverified(refresh, claims)
	if err != nil {
		return ""
	}
	return claims.ID
}

// newJTI returns a cryptographically random hex string (never math/rand).
func newJTI() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand should not fail; fall back deterministically is unsafe,
		// so panic is acceptable per crypto/rand contract.
		panic(fmt.Sprintf("token: crypto/rand read: %v", err))
	}
	return hex.EncodeToString(b)
}
