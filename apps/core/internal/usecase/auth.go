package usecase

import (
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

// TokenService issues and verifies signed JWT access/refresh pairs and
// persists refresh JTIs through a RefreshStore.
type TokenService struct {
	secret       string
	accessTTL    time.Duration
	refreshTTL   time.Duration
	refreshStore repository.RefreshStore
}

// NewTokenService builds a TokenService.
//
// The expiresAt *time.Time parameter of GenerateTokenPair is reserved (per-user
// expiry is enforced server-side in a later task) and currently ignored; the
// signature is kept for forward compatibility.
func NewTokenService(secret string, accessTTL, refreshTTL time.Duration, store repository.RefreshStore) *TokenService {
	return &TokenService{
		secret:       secret,
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		refreshStore: store,
	}
}

// GenerateTokenPair issues a signed access JWT and a signed refresh JWT for
// userID. The refresh jti is persisted in the RefreshStore. The expiresAt
// argument is reserved and ignored.
func (s *TokenService) GenerateTokenPair(userID string, _ *time.Time) (access, refresh string, err error) {
	now := time.Now()

	access, err = s.signToken(userID, claimTypeAccess, s.accessTTL, now)
	if err != nil {
		return "", "", err
	}

	refresh, err = s.signToken(userID, claimTypeRefresh, s.refreshTTL, now)
	if err != nil {
		return "", "", err
	}

	// Refresh jti is persisted so the auth usecase can verify presence.
	if err := s.refreshStore.StoreRefresh(refreshJTIFrom(refresh), userID, s.refreshTTL); err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// VerifyAccess verifies a signed access JWT (correct secret, HS256, valid
// typ, not expired) and returns the subject userID.
func (s *TokenService) VerifyAccess(token string) (string, error) {
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
func (s *TokenService) VerifyRefresh(token string) (userID, jti string, err error) {
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
func (s *TokenService) StoreRefresh(jti, userID string, ttl time.Duration) error {
	return s.refreshStore.StoreRefresh(jti, userID, ttl)
}

// ConsumeRefresh returns the userID for a stored refresh jti. See
// repository.RefreshStore.
func (s *TokenService) ConsumeRefresh(jti string) (string, bool) {
	return s.refreshStore.ConsumeRefresh(jti)
}

// DeleteRefresh removes a stored refresh jti. See repository.RefreshStore.
func (s *TokenService) DeleteRefresh(jti string) error {
	return s.refreshStore.DeleteRefresh(jti)
}

func (s *TokenService) signToken(userID, typ string, ttl time.Duration, now time.Time) (string, error) {
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

func (s *TokenService) parse(token, wantTyp string) (*tokenClaims, error) {
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
