package usecase

import (
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// fakeRefreshStore is a concurrent-safe in-memory RefreshStore for token service tests.
type fakeRefreshStore struct {
	mu sync.Mutex
	m  map[string]string
}

func newFakeRefreshStore() *fakeRefreshStore {
	return &fakeRefreshStore{m: make(map[string]string)}
}

func (f *fakeRefreshStore) StoreRefresh(jti, userID string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[jti] = userID
	return nil
}

func (f *fakeRefreshStore) ConsumeRefresh(jti string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.m[jti]
	return v, ok
}

func (f *fakeRefreshStore) DeleteRefresh(jti string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.m, jti)
	return nil
}

func TestTokenService_RoundTrip(t *testing.T) {
	ts := NewTokenService("secret", 15*time.Minute, time.Hour, newFakeRefreshStore())

	access, refresh, err := ts.GenerateTokenPair("u1", nil)
	require.NoError(t, err)
	require.NotEmpty(t, access)
	require.NotEmpty(t, refresh)

	uid, err := ts.VerifyAccess(access)
	require.NoError(t, err)
	require.Equal(t, "u1", uid)

	// wrong secret fails
	bad := NewTokenService("other", time.Minute, time.Minute, newFakeRefreshStore())
	_, err = bad.VerifyAccess(access)
	require.Error(t, err)
}

func TestTokenService_RefreshClaims(t *testing.T) {
	store := newFakeRefreshStore()
	ts := NewTokenService("s", time.Minute, time.Hour, store)

	_, refresh, err := ts.GenerateTokenPair("u2", nil)
	require.NoError(t, err)

	type withTyp struct {
		jwt.RegisteredClaims
		Typ string `json:"typ"`
	}
	claims := &withTyp{}
	_, err = jwt.ParseWithClaims(refresh, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte("s"), nil
	})
	require.NoError(t, err)

	// jti must be present
	require.NotEmpty(t, claims.ID, "jti must be present")
	require.Equal(t, "u2", claims.Subject)
	// custom typ claim distinguishes refresh tokens
	require.Equal(t, "refresh", claims.Typ)

	// refresh jti must have been persisted
	uid, ok := ts.ConsumeRefresh(claims.ID)
	require.True(t, ok)
	require.Equal(t, "u2", uid)

	// parsing the refresh token with the WRONG secret must fail
	badClaims := &withTyp{}
	_, err = jwt.ParseWithClaims(refresh, badClaims, func(t *jwt.Token) (interface{}, error) {
		return []byte("wrong"), nil
	})
	require.Error(t, err)
}

func TestTokenService_AccessExpiryRejects(t *testing.T) {
	store := newFakeRefreshStore()
	ts := NewTokenService("secret", 1*time.Nanosecond, time.Hour, store)

	access, _, err := ts.GenerateTokenPair("u3", nil)
	require.NoError(t, err)

	// Give the 1ns token a moment to expire.
	time.Sleep(2 * time.Nanosecond)

	_, err = ts.VerifyAccess(access)
	require.Error(t, err)
}

func TestNewJTI_RandomAndUnique(t *testing.T) {
	a := newJTI()
	b := newJTI()
	require.NotEmpty(t, a)
	require.Len(t, a, 32) // 16 bytes -> 32 hex chars
	require.NotEqual(t, a, b, "two JTIs must differ")
}

func TestTokenService_VerifyRejectsWrongType(t *testing.T) {
	store := newFakeRefreshStore()
	ts := NewTokenService("secret", time.Minute, time.Hour, store)

	_, refresh, err := ts.GenerateTokenPair("u4", nil)
	require.NoError(t, err)

	// Verifying a refresh token as an access token must fail (typ mismatch).
	_, err = ts.VerifyAccess(refresh)
	require.Error(t, err)
}

var _ repository.RefreshStore = (*fakeRefreshStore)(nil)
