package token

import (
	"context"
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

func (f *fakeRefreshStore) StoreRefresh(_ context.Context, jti, userID string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[jti] = userID
	return nil
}

func (f *fakeRefreshStore) ConsumeRefresh(_ context.Context, jti string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.m[jti]
	return v, ok
}

func (f *fakeRefreshStore) DeleteRefresh(_ context.Context, jti string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.m, jti)
	return nil
}

var _ repository.RefreshStore = (*fakeRefreshStore)(nil)

func TestTokenService(t *testing.T) {
	tests := []struct {
		name    string
		actions func(t *testing.T, ts *JWTTokenService)
	}{
		{
			name: "round trip generate and verify",
			actions: func(t *testing.T, ts *JWTTokenService) {
				access, refresh, err := ts.Sign(context.Background(), "u1", nil)
				require.NoError(t, err)
				require.NotEmpty(t, access)
				require.NotEmpty(t, refresh)

				uid, err := ts.VerifyAccess(access)
				require.NoError(t, err)
				require.Equal(t, "u1", uid)
			},
		},
		{
			name: "wrong secret fails verification",
			actions: func(t *testing.T, ts *JWTTokenService) {
				access, _, err := ts.Sign(context.Background(), "u1", nil)
				require.NoError(t, err)

				bad := NewJWTTokenService("other", time.Minute, time.Minute, newFakeRefreshStore())
				_, err = bad.VerifyAccess(access)
				require.Error(t, err)
			},
		},
		{
			name: "expired token rejected",
			actions: func(t *testing.T, ts *JWTTokenService) {
				shortLived := NewJWTTokenService("secret", 1*time.Nanosecond, time.Hour, newFakeRefreshStore())
				access, _, err := shortLived.Sign(context.Background(), "u3", nil)
				require.NoError(t, err)

				time.Sleep(2 * time.Nanosecond)
				_, err = shortLived.VerifyAccess(access)
				require.Error(t, err)
			},
		},
		{
			name: "refresh token has correct claims",
			actions: func(t *testing.T, ts *JWTTokenService) {
				_, refresh, err := ts.Sign(context.Background(), "u2", nil)
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
				require.NotEmpty(t, claims.ID, "jti must be present")
				require.Equal(t, "u2", claims.Subject)
				require.Equal(t, "refresh", claims.Typ)

				uid, ok := ts.ConsumeRefresh(context.Background(), claims.ID)
				require.True(t, ok)
				require.Equal(t, "u2", uid)
			},
		},
		{
			name: "verify rejects refresh token as access",
			actions: func(t *testing.T, ts *JWTTokenService) {
				_, refresh, err := ts.Sign(context.Background(), "u4", nil)
				require.NoError(t, err)

				_, err = ts.VerifyAccess(refresh)
				require.Error(t, err)
			},
		},
		{
			name: "jti is random and unique",
			actions: func(t *testing.T, ts *JWTTokenService) {
				a := newJTI()
				b := newJTI()
				require.NotEmpty(t, a)
				require.Len(t, a, 32) // 16 bytes -> 32 hex chars
				require.NotEqual(t, a, b, "two JTIs must differ")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewJWTTokenService("s", time.Minute, time.Hour, newFakeRefreshStore())
			tt.actions(t, ts)
		})
	}
}
